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
	"context"
	"log"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	clientset "github.com/grpc/test-infra/clientset"
)

// AfterIntervalFunction returns a function that stops for a time interval.
// This function is provided so it can be replaced with a fake for testing.
func AfterIntervalFunction(d time.Duration) func() {
	return func() {
		<-time.After(d)
	}
}

// Runner contains the information needed to run multiple sets of LoadTests.
type Runner struct {
	// loadTestGetter interacts with the cluster to create, get and delete
	// LoadTests.
	loadTestGetter clientset.LoadTestGetter
	// afterInterval stops for a set time interval before returning.
	// It is used to set a polling interval.
	afterInterval func()
	// retries is the number of times to retry create and poll operations before
	// failing each test.
	retries uint
	// deleteSuccessfulTests determines whether tests that terminate without
	// errors should be deleted immediately.
	deleteSuccessfulTests bool
	// podLogger stores pod log files
	podLogger *PodLogger
}

// NewRunner creates a new Runner object.
func NewRunner(loadTestGetter clientset.LoadTestGetter, afterInterval func(), retries uint, deleteSuccessfulTests bool) *Runner {
	return &Runner{
		loadTestGetter:        loadTestGetter,
		afterInterval:         afterInterval,
		retries:               retries,
		deleteSuccessfulTests: deleteSuccessfulTests,
		podLogger:             NewPodLogger(),
	}
}

// Run runs a set of LoadTests at a given concurrency level.
func (r *Runner) Run(ctx context.Context, configs []*grpcv1.LoadTest, suiteReporter *TestSuiteReporter, concurrencyLevel int, podLogDir string, done chan<- *TestSuiteReporter) {
	var count, n int
	qName := suiteReporter.Queue()
	testDone := make(chan *TestCaseReporter)
	for _, config := range configs {
		for n >= concurrencyLevel {
			reporter := <-testDone
			reporter.SetEndTime(time.Now())
			log.Printf("Finished test in queue %s after %v", qName, reporter.Duration())
			n--
			count++
			log.Printf("Finished %d tests in queue %s", count, qName)
		}
		n++
		reporter := suiteReporter.NewTestCaseReporter(config)
		log.Printf("Starting test %d in queue %s", reporter.Index(), qName)
		reporter.SetStartTime(time.Now())
		go r.runTest(ctx, config, reporter, podLogDir, testDone)
	}
	for n > 0 {
		reporter := <-testDone
		reporter.SetEndTime(time.Now())
		log.Printf("Finished test in queue %s after %v", qName, reporter.Duration())
		n--
		count++
		log.Printf("Finished %d tests in queue %s", count, qName)
	}
	done <- suiteReporter
}

// runTest creates a single LoadTest and monitors it to completion.
func (r *Runner) runTest(ctx context.Context, config *grpcv1.LoadTest, reporter *TestCaseReporter, podLogDir string, done chan<- *TestCaseReporter) {
	var s, status string
	var retries uint

	for {
		loadTest, err := r.loadTestGetter.Create(ctx, config, metav1.CreateOptions{})
		if err != nil {
			reporter.Warning("Failed to create test %s: %v", config.Name, err)
			if retries < r.retries {
				retries++
				reporter.Info("Scheduling retry %d/%d to create test", retries, r.retries)
				r.afterInterval()
				continue
			}
			reporter.Error("Aborting after %d retries to create test %s: %v", r.retries, config.Name, err)
			done <- reporter
			return
		}
		retries = 0
		config.Status = loadTest.Status
		reporter.Info("Created test %s", config.Name)
		break
	}

	for {
		loadTest, err := r.loadTestGetter.Get(ctx, config.Name, metav1.GetOptions{})
		if err != nil {
			reporter.Warning("Failed to poll test %s: %v", config.Name, err)
			if retries < r.retries {
				retries++
				reporter.Info("Scheduling retry %d/%d to poll test", retries, r.retries)
				r.afterInterval()
				continue
			}
			reporter.Error("Aborting test after %d retries to poll test %s: %v", r.retries, config.Name, err)
			done <- reporter
			return
		}
		retries = 0
		config.Status = loadTest.Status
		s = status
		status = statusString(config)
		switch {
		case loadTest.Status.State.IsTerminated():
			err = r.podLogger.savePodLogs(ctx, loadTest, podLogDir)
			if err != nil {
				reporter.Error("Could not save pod logs: %s", err)
			}

			if status != "Succeeded" {
				reporter.Error("Test failed with reason %q: %v", loadTest.Status.Reason, loadTest.Status.Message)
			} else {
				reporter.Info("Test terminated with a status of %q", status)
				if r.deleteSuccessfulTests {
					err = r.loadTestGetter.Delete(ctx, config.Name, metav1.DeleteOptions{})
					if err != nil {
						reporter.Info("Failed to delete test %s: %v", config.Name, err)
					} else {
						reporter.Info("Deleted test %s", config.Name)
					}
				}
			}
			done <- reporter
			return
		case loadTest.Status.State == grpcv1.Running:
			reporter.Info("%s", status)
			r.afterInterval()
		default:
			if s != status {
				reporter.Info("%s", status)
			}
			// Use a longer polling interval for tests that have not started.
			r.afterInterval()
			r.afterInterval()
		}
	}
}

// statusString returns a string to represent the test status in logs.
// The string consists of state, reason and message (each omitted if empty).
func statusString(config *grpcv1.LoadTest) string {
	s := []string{string(config.Status.State)}
	if reason := strings.TrimSpace(config.Status.Reason); reason != "" {
		s = append(s, reason)
	}
	if message := strings.TrimSpace(config.Status.Message); message != "" {
		s = append(s, message)
	}
	return strings.Join(s, "; ")
}
