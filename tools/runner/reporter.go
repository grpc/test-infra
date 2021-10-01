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

package runner

import (
	"fmt"
	"log"
	"strings"
	"time"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/tools/runner/xunit"
)

// Reporter instances log the progress of the test suites and cases, filling a
// xunit.Report instance if provided.
type Reporter struct {
	report    *xunit.Report
	startTime time.Time
	endTime   time.Time
}

// NewReporter constructs a new reporter instance.
func NewReporter(report *xunit.Report) *Reporter {
	return &Reporter{report: report}
}

// SetStartTime records the start time for the test suites as a whole.
func (r *Reporter) SetStartTime(t time.Time) {
	r.startTime = t
}

// SetEndTime records the end time for the test suites as a whole.
func (r *Reporter) SetEndTime(t time.Time) {
	r.endTime = t

	if r.report == nil {
		return
	}
	r.report.TimeInSeconds = t.Sub(r.startTime).Seconds()
}

// Duration returns the elapsed time between the time.Time instances passed to
// the SetStartTime and SetEndTime methods. Ideally, these should be used at the
// beginning and end of running all test suites to produce the wall-clock time.
// If these values are not set, a zero value is returned.
func (r *Reporter) Duration() time.Duration {
	if r.startTime.IsZero() || r.endTime.IsZero() {
		return 0
	}

	return r.endTime.Sub(r.startTime)
}

// NewTestSuiteReporter creates a new suite reporter instance.
func (r *Reporter) NewTestSuiteReporter(qName string, logPrefixFmt string, testCaseName func(*grpcv1.LoadTest) string) *TestSuiteReporter {
	suiteReporter := &TestSuiteReporter{
		qName:        qName,
		logPrefixFmt: logPrefixFmt,
		testCaseName: testCaseName,
	}

	if r.report != nil {
		testSuite := &xunit.TestSuite{
			Name: qName,
		}
		r.report.Suites = append(r.report.Suites, testSuite)
		suiteReporter.testSuite = testSuite
	}

	return suiteReporter
}

// TestSuiteReporter manages reports for tests that share a runner queue.
type TestSuiteReporter struct {
	testSuite    *xunit.TestSuite
	testCount    int
	qName        string
	logPrefixFmt string
	testCaseName func(*grpcv1.LoadTest) string
	startTime    time.Time
	endTime      time.Time
}

// Queue returns the name of the queue containing tests for this test suite.
func (tsr *TestSuiteReporter) Queue() string {
	return tsr.qName
}

// SetStartTime records the start time of the test suite.
func (tsr *TestSuiteReporter) SetStartTime(t time.Time) {
	tsr.startTime = t
}

// SetEndTime records the end time of the test suite.
func (tsr *TestSuiteReporter) SetEndTime(t time.Time) {
	tsr.endTime = t

	if tsr.testSuite == nil {
		return
	}
	tsr.testSuite.TimeInSeconds = tsr.Duration().Seconds()
}

// Duration returns the elapsed time between the time.Time instances passed to
// the SetStartTime and SetEndTime methods. Ideally, these should be used at the
// beginning and end of the test suite to produce the wall-clock time. If these
// values are not set, a zero value is returned.
func (tsr *TestSuiteReporter) Duration() time.Duration {
	if tsr.startTime.IsZero() || tsr.endTime.IsZero() {
		return 0
	}

	return tsr.endTime.Sub(tsr.startTime)
}

// NewTestCaseReporter creates a new reporter instance.
func (tsr *TestSuiteReporter) NewTestCaseReporter(config *grpcv1.LoadTest) *TestCaseReporter {
	index := tsr.testCount
	tsr.testCount++

	logPrefix := fmt.Sprintf(tsr.logPrefixFmt, tsr.qName, index)
	caseReporter := &TestCaseReporter{
		logPrintf: func(format string, v ...interface{}) {
			log.Printf(logPrefix+format, v...)
		},
		index: index,
	}

	if tsr.testSuite != nil {
		testCase := &xunit.TestCase{
			Name: tsr.testCaseName(config),
		}
		tsr.testSuite.Cases = append(tsr.testSuite.Cases, testCase)
		caseReporter.testCase = testCase
	}

	return caseReporter
}

// TestCaseReporter collects events for logging and reporting during a test.
type TestCaseReporter struct {
	testCase  *xunit.TestCase
	logPrintf func(format string, v ...interface{})
	index     int
	startTime time.Time
	endTime   time.Time
}

// Index returns the index of the test case in the test suite (and queue).
func (tcr *TestCaseReporter) Index() int {
	return tcr.index
}

// Info records an informational message generated by the test.
func (tcr *TestCaseReporter) Info(format string, v ...interface{}) {
	tcr.logPrintf(format, v...)
}

// Warning records a warning message generated during the test.
// The error that caused the message to be generated is also included.
func (tcr *TestCaseReporter) Warning(format string, v ...interface{}) {
	tcr.logPrintf(format, v...)
}

// Error records an error message generated during the test.
// The error that caused the message to be generated is also included.
func (tcr *TestCaseReporter) Error(format string, v ...interface{}) {
	tcr.logPrintf(format, v...)

	if tcr.testCase == nil {
		return
	}
	tcr.testCase.Errors = append(tcr.testCase.Errors, &xunit.Error{
		Message: fmt.Sprintf(format, v...),
	})
}

// SetStartTime records the start time of the test.
func (tcr *TestCaseReporter) SetStartTime(t time.Time) {
	tcr.startTime = t
}

// SetEndTime records the end time of the test.
func (tcr *TestCaseReporter) SetEndTime(t time.Time) {
	tcr.endTime = t

	if tcr.testCase == nil {
		return
	}
	tcr.testCase.TimeInSeconds = tcr.Duration().Seconds()
}

// Duration returns the elapsed time between the time.Time instances passed to
// the SetStartTime and SetEndTime methods. Ideally, these should be used at the
// beginning and end of the test to produce the wall-clock time. If these values
// are not set, a zero value is returned.
func (tcr *TestCaseReporter) Duration() time.Duration {
	if tcr.startTime.IsZero() || tcr.endTime.IsZero() {
		return 0
	}

	return tcr.endTime.Sub(tcr.startTime)
}

// AddProperty adds a key-value property to the test case.
func (tcr *TestCaseReporter) AddProperty(key, value string) {
	property := &xunit.Property{
		Key:   key,
		Value: value,
	}
	tcr.testCase.Properties = append(tcr.testCase.Properties, property)
}

// TestCaseNameFromAnnotations returns a function to generate test case names.
// Test case names are derived from the value of annotations added to the test
// configuration.
func TestCaseNameFromAnnotations(annotationKeys ...string) func(*grpcv1.LoadTest) string {
	return func(config *grpcv1.LoadTest) string {
		var values []string
		for _, key := range annotationKeys {
			if value := config.Annotations[key]; value != "" {
				values = append(values, strings.ToLower(xunit.Dashify(value)))
			}
		}
		return strings.Join(values, "-")
	}
}
