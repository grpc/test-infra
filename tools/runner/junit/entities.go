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

package junit

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// Report encapsulates the data for a JUnit XML report.
type Report struct {
	XMLName       xml.Name     `xml:"testsuites"`
	ID            string       `xml:"id,attr"`
	Name          string       `xml:"name,attr"`
	TestCount     int          `xml:"tests,attr"`
	FailureCount  int          `xml:"failures,attr"`
	TimeInSeconds float64      `xml:"time,attr"`
	Suites        []*TestSuite `xml:"testsuite"`
}

// DeepCopy makes an exact copy of a Report object and all child objects.
func (r *Report) DeepCopy() *Report {
	c := &Report{
		ID:            r.ID,
		Name:          r.Name,
		TestCount:     r.TestCount,
		FailureCount:  r.FailureCount,
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

	for _, testSuite := range c.Suites {
		testSuite.FailureCount = 0
		testSuite.TestCount = len(testSuite.Cases)
		for _, testCase := range testSuite.Cases {
			testSuite.FailureCount += len(testCase.Failures)
		}

		c.FailureCount += testSuite.FailureCount
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
	FailureCount  int         `xml:"failures,attr"`
	TimeInSeconds float64     `xml:"time,attr"`
	Cases         []*TestCase `xml:"testcase"`
}

// DeepCopy makes an exact copy of a TestSuite object and all child objects.
func (ts *TestSuite) DeepCopy() *TestSuite {
	c := &TestSuite{
		ID:            ts.ID,
		Name:          ts.Name,
		TestCount:     ts.TestCount,
		FailureCount:  ts.FailureCount,
		TimeInSeconds: ts.TimeInSeconds,
	}
	for _, tc := range ts.Cases {
		c.Cases = append(c.Cases, tc.DeepCopy())
	}
	return c
}

// TestCase encapsulates metadata regarding a single test.
type TestCase struct {
	XMLName       xml.Name   `xml:"testcase"`
	ID            string     `xml:"id,attr"`
	Name          string     `xml:"name,attr"`
	TimeInSeconds float64    `xml:"time,attr"`
	Failures      []*Failure `xml:"failure"`
}

// DeepCopy makes an exact copy of a TestCase object and all child objects.
func (tc *TestCase) DeepCopy() *TestCase {
	c := &TestCase{
		ID:            tc.ID,
		Name:          tc.Name,
		TimeInSeconds: tc.TimeInSeconds,
	}
	for _, f := range tc.Failures {
		c.Failures = append(c.Failures, f.DeepCopy())
	}
	return c
}

// FailureType is a possible value of the "type" attribute on the
// Failure JUnit XML tag.
type FailureType string

const (
	// Warning signals that an error occurred which is not fatal.
	Warning FailureType = "warning"

	// Error signals that an error occurred which is fatal.
	Error FailureType = "error"
)

// Failure encapsulates metadata regarding a test failure or warning.
type Failure struct {
	XMLName xml.Name    `xml:"failure"`
	Type    FailureType `xml:"type,attr"`
	Message string      `xml:"message,attr"`
	Text    string      `xml:",chardata"`
}

// DeepCopy makes an exact copy of a Failure object and all child objects.
func (f *Failure) DeepCopy() *Failure {
	return &Failure{
		Type:    f.Type,
		Message: f.Message,
		Text:    f.Text,
	}
}
