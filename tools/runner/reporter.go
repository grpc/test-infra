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

// Package runner contains code for a test runner that can run a list of
// load tests, wait for them to complete, and report on the results.
package runner

import (
	"encoding/xml"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/inspect"
	"github.com/grpc/test-infra/tools/runner/junit"
	"github.com/pkg/errors"
)

// TestSuitesReporter manages reports for all groups of tests.
type TestSuitesReporter struct {
	tss         *junit.TestSuites
	tsReporters []*TestSuiteReporter
	startTime   time.Time
}

func NewTestSuitesReporter(name string) *TestSuitesReporter {
	return &TestSuitesReporter{
		tss: &junit.TestSuites{
			ID:   Dashify(name),
			Name: name,
		},
	}
}

// SetStartTime records the start time of the test.
func (r *TestSuitesReporter) SetStartTime(startTime time.Time) {
	r.startTime = startTime
}

// SetEndTime records the end time of the test.
func (r *TestSuitesReporter) SetEndTime(endTime time.Time) {
	r.tss.TimeInSeconds = endTime.Sub(r.startTime).Seconds()
}

// Failures returns the number of failures that the test case experienced.
func (r *TestSuitesReporter) Failures() int {
	failures := 0
	for _, tsReporter := range r.tsReporters {
		failures += tsReporter.Failures()
	}
	return failures
}

func (r *TestSuitesReporter) TestCount() int {
	testCount := 0
	for _, tsReporter := range r.tsReporters {
		testCount += tsReporter.TestCount()
	}
	return testCount
}

func (r *TestSuitesReporter) junitObject() *junit.TestSuites {
	r.tss.Suites = []*junit.TestSuite{}
	for _, tsReporter := range r.tsReporters {
		r.tss.Suites = append(r.tss.Suites, tsReporter.junitObject())
	}
	r.tss.FailureCount = r.Failures()
	r.tss.TestCount = r.TestCount()
	return r.tss
}

// NewTestSuiteReporter creates a new suite reporter instance.
func (r *TestSuitesReporter) NewTestSuiteReporter(qName string, logPrefixFmt string) *TestSuiteReporter {
	tsReporter := &TestSuiteReporter{
		qName:        qName,
		logPrefixFmt: logPrefixFmt,
		ts: &junit.TestSuite{
			ID:   Dashify(qName),
			Name: qName,
		},
	}
	r.tsReporters = append(r.tsReporters, tsReporter)
	return tsReporter
}

func (r *TestSuitesReporter) XMLReport() ([]byte, error) {
	tssJUnitObject := r.junitObject()
	bytes, err := xml.MarshalIndent(tssJUnitObject, "", "  ")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate an XML report")
	}
	return bytes, nil
}

// TestSuiteReporter manages reports for tests that share a runner queue.
type TestSuiteReporter struct {
	qName        string
	logPrefixFmt string
	ts           *junit.TestSuite
	tcReporters  []*TestCaseReporter
	startTime    time.Time
}

// SetStartTime records the start time of the test.
func (sr *TestSuiteReporter) SetStartTime(startTime time.Time) {
	sr.startTime = startTime
}

// SetEndTime records the end time of the test.
func (sr *TestSuiteReporter) SetEndTime(endTime time.Time) {
	sr.ts.TimeInSeconds = endTime.Sub(sr.startTime).Seconds()
}

// Queue returns the name of the queue containing tests for this test suite.
func (sr *TestSuiteReporter) Queue() string {
	return sr.qName
}

// Failures returns the number of failures that the test case experienced.
func (sr *TestSuiteReporter) Failures() int {
	failures := 0
	for _, tcReporter := range sr.tcReporters {
		failures += tcReporter.Failures()
	}
	return failures
}

func (sr *TestSuiteReporter) TestCount() int {
	return len(sr.tcReporters)
}

func (sr *TestSuiteReporter) junitObject() *junit.TestSuite {
	sr.ts.Cases = []*junit.TestCase{}
	for _, tcReporter := range sr.tcReporters {
		sr.ts.Cases = append(sr.ts.Cases, tcReporter.junitObject())
	}
	sr.ts.FailureCount = sr.Failures()
	sr.ts.TestCount = sr.TestCount()
	return sr.ts
}

// NewTestCaseReporter creates a new reporter instance.
func (sr *TestSuiteReporter) NewTestCaseReporter(config *grpcv1.LoadTest) *TestCaseReporter {
	index := len(sr.tcReporters)
	logPrefix := fmt.Sprintf(sr.logPrefixFmt, sr.qName, index)

	tc := &junit.TestCase{}
	tcReporter := &TestCaseReporter{
		logPrintf: func(format string, v ...interface{}) {
			log.Printf(logPrefix+format, v...)
		},
		index: index,
		tc:    tc,
	}
	sr.tcReporters = append(sr.tcReporters, tcReporter)

	tc.ID = config.Name
	scenarioNames, err := inspect.GetScenarioNames(config)
	if err == nil {
		tc.Name = strings.Join(scenarioNames, ",")
	} else {
		tcReporter.Warning("Failed to find scenario name", "Failed to extract a name for the test case from the .spec.scenariosJSON field: %v", err.Error())
		tc.Name = "scenario_name_not_found"
	}

	return tcReporter
}

// TestCaseReporter collects events for logging and reporting during a test.
type TestCaseReporter struct {
	tc        *junit.TestCase
	startTime time.Time
	logPrintf func(format string, v ...interface{})
	index     int
}

// Index returns the index of the test case in the test suite (and queue).
func (cr *TestCaseReporter) Index() int {
	return cr.index
}

// Failures returns the number of failures that the test case experienced.
func (cr *TestCaseReporter) Failures() int {
	return len(cr.tc.Failures)
}

// Info records an informational message generated by the test. This information
// is included in the logs, but it is not included in the JUnit report.
func (cr *TestCaseReporter) Info(format string, v ...interface{}) {
	cr.logPrintf(format, v...)
}

// Warning records a warning message generated during the test. The error that
// caused the message to be generated is also included. It accepts a message
// which is a short string literal that identifies the problem, as well as, a
// format string and arguments for a more verbose description.
func (cr *TestCaseReporter) Warning(message, textFmt string, v ...interface{}) {
	cr.logPrintf(textFmt, v...)
	cr.tc.Failures = append(cr.tc.Failures, &junit.Failure{
		Type:    junit.Warning,
		Message: message,
		Text:    fmt.Sprintf(textFmt, v...),
	})
}

// Error records an error message generated during the test. The error that
// caused the message to be generated is also included. It accepts a message
// which is a short string literal that identifies the problem, as well as, a
// format string and arguments for a more verbose description.
func (cr *TestCaseReporter) Error(message, textFmt string, v ...interface{}) {
	cr.logPrintf(textFmt, v...)
	cr.tc.Failures = append(cr.tc.Failures, &junit.Failure{
		Type:    junit.Error,
		Message: message,
		Text:    fmt.Sprintf(textFmt, v...),
	})
}

// SetStartTime records the start time of the test.
func (cr *TestCaseReporter) SetStartTime(startTime time.Time) {
	cr.startTime = startTime
}

// SetEndTime records the end time of the test.
func (cr *TestCaseReporter) SetEndTime(endTime time.Time) {
	cr.tc.TimeInSeconds = endTime.Sub(cr.startTime).Seconds()
}

func (cr *TestCaseReporter) TestDuration() time.Duration {
	return time.Duration(cr.tc.TimeInSeconds)
}

func (cr *TestCaseReporter) junitObject() *junit.TestCase {
	return cr.tc
}

// Dashify returns the input string where all whitespace and underscore
// characters have been replaced by dashes and, aside from dashes, only
// alphanumeric characters remain.
func Dashify(str string) string {
	// TODO: Move this into another shared package.
	b := strings.Builder{}
	for _, rune := range str {
		if string(rune) == "_" || unicode.IsSpace(rune) {
			b.WriteString("-")
		} else if string(rune) == "-" || unicode.IsLetter(rune) || unicode.IsNumber(rune) {
			b.WriteRune(rune)
		}
	}
	return b.String()
}
