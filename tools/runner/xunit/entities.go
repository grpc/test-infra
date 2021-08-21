/*
Copyright 2021 gRPC authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package xunit

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// Report encapsulates the data for a xUnit XML report.
type Report struct {
	XMLName       xml.Name     `xml:"testsuites"`
	Name          string       `xml:"name,attr"`
	TestCount     int          `xml:"tests,attr"`
	ErrorCount    int          `xml:"errors,attr"`
	TimeInSeconds float64      `xml:"time,attr"`
	Suites        []*TestSuite `xml:"testsuite"`
}

// Finalize iterates over the document object model and recomputes the counter
// values for parent objects. This ensures, for instance, that the errors
// attribute of a test suite specifies the correct sum of errors from its child
// test cases. This method should be called once all test cases are complete.
func (r *Report) Finalize() {
	r.TestCount = 0

	for i, testSuite := range r.Suites {
		testSuite.ID = fmt.Sprint(i)
		testSuite.ErrorCount = 0
		testSuite.TestCount = len(testSuite.Cases)
		for _, testCase := range testSuite.Cases {
			testSuite.ErrorCount += len(testCase.Errors)
		}

		r.ErrorCount += testSuite.ErrorCount
		r.TestCount += testSuite.TestCount
	}
}

// Split separates each test suite into a separate XML report.
// The reports are returned as a map of test suite names to XML reports, where
// each report contains a single test suite.
func (r *Report) Split() map[string]*Report {
	m := make(map[string]*Report)
	for _, testSuite := range r.Suites {
		report := &Report{
			Name:          testSuite.Name,
			TimeInSeconds: testSuite.TimeInSeconds,
			Suites:        []*TestSuite{testSuite},
		}
		report.Finalize()
		m[testSuite.Name] = report
	}
	return m
}

// ReportWritingOptions wraps optional settings for the output report.
type ReportWritingOptions struct {
	// Number of spaces which should be used for indentation.
	IndentSize int

	// Number of times to retry if writing to a stream fails and no progress is
	// being made on each retry.
	MaxRetries int
}

// WriteToStream accepts any io.Writer and writes the contents of the report to
// the stream. It accepts a ReportWritingOptions instance, which provides
// additional granularity for tweaking the output. The method r.Finalize()
// should be called before writing the report.
func (r *Report) WriteToStream(w io.Writer, opts ReportWritingOptions) error {
	bytes, err := xml.MarshalIndent(r, "", strings.Repeat(" ", opts.IndentSize))
	if err != nil {
		return errors.Wrapf(err, "failed to write xUnit report to stream")
	}
	bytes = append(bytes, '\n')

	for n, prevN, retries := 0, 0, 0; n < len(bytes); {
		n, err = w.Write(bytes[n:])
		if err != nil {
			if n == prevN && retries >= opts.MaxRetries {
				return errors.Wrapf(err, "failed to write %d bytes of xUnit report to stream", len(bytes)-n)
			}

			prevN = n
			retries++
		}
	}

	return nil
}

// TestSuite encapsulates metadata for a collection of test cases.
type TestSuite struct {
	XMLName       xml.Name    `xml:"testsuite"`
	ID            string      `xml:"id,attr"`
	Name          string      `xml:"name,attr"`
	TestCount     int         `xml:"tests,attr"`
	ErrorCount    int         `xml:"errors,attr"`
	TimeInSeconds float64     `xml:"time,attr"`
	Cases         []*TestCase `xml:"testcase"`
}

// TestCase encapsulates metadata regarding a single test.
type TestCase struct {
	XMLName       xml.Name `xml:"testcase"`
	Name          string   `xml:"name,attr"`
	TimeInSeconds float64  `xml:"time,attr"`
	Errors        []*Error `xml:"error"`
}

// Error encapsulates metadata regarding a test error.
type Error struct {
	XMLName xml.Name `xml:"error"`
	Message string   `xml:"message,attr,omitempty"`
	Text    string   `xml:",chardata"`
}
