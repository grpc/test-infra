/*
Copyright 2020 gRPC authors.

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

import "encoding/xml"

// TestSuites is the top-level entity in a JUnit XML report. It encapsulates
// the metadata regarding all of the test suites.
type TestSuites struct {
	XMLName       xml.Name     `xml:"testsuites"`
	ID            string       `xml:"id,attr"`
	Name          string       `xml:"name,attr"`
	TestCount     int          `xml:"tests,attr"`
	FailureCount  int          `xml:"failures,attr"`
	TimeInSeconds float64      `xml:"time,attr"`
	Suites        []*TestSuite `xml:"testsuite"`
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

// TestCase encapsulates metadata regarding a single test.
type TestCase struct {
	XMLName       xml.Name   `xml:"testcase"`
	ID            string     `xml:"id,attr"`
	Name          string     `xml:"name,attr"`
	TimeInSeconds float64    `xml:"time,attr"`
	Failures      []*Failure `xml:"failure"`
}

// FailureType is a possible value of the "type" attribute on the
// Failure jUnit XML tag.
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
