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

// Report encapsulates the data for a JUnit XML report.
type Report struct {
	XMLName       xml.Name     `xml:"testsuites"`
	Name          string       `xml:"name,attr"`
	TestCount     int          `xml:"tests,attr"`
	ErrorCount    int          `xml:"errors,attr"`
	TimeInSeconds float64      `xml:"time,attr"`
	Suites        []*TestSuite `xml:"testsuite"`
}

// DeepCopy makes an exact copy of a Report object and all child objects.
func (r *Report) DeepCopy() *Report {
	c := &Report{
		Name:          r.Name,
		TestCount:     r.TestCount,
		ErrorCount:    r.ErrorCount,
		TimeInSeconds: r.TimeInSeconds,
	}
	for _, ts := range r.Suites {
		c.Suites = append(c.Suites, ts.DeepCopy())
	}
	return c
}

// Finalize makes a deep copy of the report. Then, it recursively maps over the
// document object model and recomputes the counter values for parent objects.
// This ensures, for instance, that the failures attribute of a test suite
// specifies the correct sum of failures from its child test cases. Finally, it
// returns the copy with the correct values.
//
// This immutability ensures that there is thread-safety between reading and
// writing to the report. This allows the report to be written even if not all
// test cases have completed without harming later reporting.
func (r *Report) Finalize() *Report {
	c := r.DeepCopy()
	c.TestCount = 0

	for i, testSuite := range c.Suites {
		testSuite.ID = fmt.Sprint(i)
		testSuite.ErrorCount = 0
		testSuite.TestCount = len(testSuite.Cases)
		for _, testCase := range testSuite.Cases {
			testSuite.ErrorCount += len(testCase.Errors)
		}

		c.ErrorCount += testSuite.ErrorCount
		c.TestCount += testSuite.TestCount
	}

	return c
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
// additional granularity for tweaking the output.
func (r *Report) WriteToStream(w io.Writer, opts ReportWritingOptions) error {
	c := r.Finalize()
	bytes, err := xml.MarshalIndent(c, "", strings.Repeat(" ", opts.IndentSize))
	if err != nil {
		return errors.Wrapf(err, "failed to write JUnit report to stream")
	}

	for n, prevN, retries := 0, 0, 0; n < len(bytes); {
		n, err = w.Write(bytes[n:])
		if err != nil {
			if n == prevN && retries >= opts.MaxRetries {
				return errors.Wrapf(err, "failed to write %d bytes of JUnit report to stream", len(bytes)-n)
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

// DeepCopy makes an exact copy of a TestSuite object and all child objects.
func (ts *TestSuite) DeepCopy() *TestSuite {
	c := &TestSuite{
		ID:            ts.ID,
		Name:          ts.Name,
		TestCount:     ts.TestCount,
		ErrorCount:    ts.ErrorCount,
		TimeInSeconds: ts.TimeInSeconds,
	}
	for _, tc := range ts.Cases {
		c.Cases = append(c.Cases, tc.DeepCopy())
	}
	return c
}

// TestCase encapsulates metadata regarding a single test.
type TestCase struct {
	XMLName       xml.Name `xml:"testcase"`
	Name          string   `xml:"name,attr"`
	TimeInSeconds float64  `xml:"time,attr"`
	Errors        []*Error `xml:"error"`
}

// DeepCopy makes an exact copy of a TestCase object and all child objects.
func (tc *TestCase) DeepCopy() *TestCase {
	c := &TestCase{
		Name:          tc.Name,
		TimeInSeconds: tc.TimeInSeconds,
	}
	for _, f := range tc.Errors {
		c.Errors = append(c.Errors, f.DeepCopy())
	}
	return c
}

// Error encapsulates metadata regarding a test error.
type Error struct {
	XMLName xml.Name `xml:"error"`
	Message string   `xml:"message,attr,omitempty"`
	Text    string   `xml:",chardata"`
}

// DeepCopy makes an exact copy of a Failure object and all child objects.
func (f *Error) DeepCopy() *Error {
	return &Error{
		Message: f.Message,
		Text:    f.Text,
	}
}
