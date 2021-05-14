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
	"fmt"
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
	// Done receives each queue name when the runner is done with that queue.
	Done chan string
	// loadTestGetter interacts with the cluster to create, get and delete
	// LoadTests.
	loadTestGetter clientset.LoadTestGetter
	// logPrefixFmt is the string used to format queue name and index into a
	// prefix when printing logs for each test.
	logPrefixFmt string
	// afterInterval is a function that waits for a set time interval before
	// returning. This function is used to set the polling interval to use
	// while running each test.
	afterInterval func()
	// retries is the number of times to retry create and poll operations before
	// failing each test.
	retries uint
}

// NewRunner creates a new Runner object.
func NewRunner(loadTestGetter clientset.LoadTestGetter, logPrefixFmt string, afterInterval func(), retries uint) *Runner {
	return &Runner{
		Done:           make(chan string),
		logPrefixFmt:   logPrefixFmt,
		loadTestGetter: loadTestGetter,
		afterInterval:  afterInterval,
		retries:        retries,
	}
}

// Run runs a set of LoadTests at a given concurrency level.
func (r *Runner) Run(qName string, configs []*grpcv1.LoadTest, concurrencyLevel int) {
	var count, n int
	done := make(chan int)
	for i, config := range configs {
		for n >= concurrencyLevel {
			<-done
			n--
			count++
			log.Printf("Finished %d tests in queue %s", count, qName)
		}
		log.Printf("Starting test %d in queue %s", i, qName)
		logPrintf := r.logPrintf(qName, i)
		n++
		go r.runTest(logPrintf, config, i, done)
	}
	for n > 0 {
		<-done
		n--
		count++
		log.Printf("Finished %d tests in queue %s", count, qName)
	}
	r.Done <- qName
}

// logPrintf returns a function to print logs for each test.
func (r *Runner) logPrintf(qName string, index int) func(string, ...interface{}) {
	logPrefixFmt := fmt.Sprintf(r.logPrefixFmt, qName, index)
	return func(format string, v ...interface{}) {
		log.Printf(logPrefixFmt+format, v...)
	}
}

// runTest creates a single LoadTest and monitors it to completion.
func (r *Runner) runTest(logPrintf func(string, ...interface{}), config *grpcv1.LoadTest, i int, done chan int) {
	name := nameString(config)
	var s, status string
	var retries uint

	for {
		loadTest, err := r.loadTestGetter.Create(config, metav1.CreateOptions{})
		if err != nil {
			logPrintf("Failed to create test %s", name)
			if retries < r.retries {
				retries++
				logPrintf("Scheduling retry %d/%d to create test", retries, r.retries)
				r.afterInterval()
				continue
			}
			logPrintf("Aborting after %d retries to create test %s", r.retries, name)
			done <- i
			return
		}
		retries = 0
		config.Status = loadTest.Status
		logPrintf("Created test %s", name)
		break
	}

	for {
		loadTest, err := r.loadTestGetter.Get(config.Name, metav1.GetOptions{})
		if err != nil {
			logPrintf("Failed to poll test %s", name)
			if retries < r.retries {
				retries++
				logPrintf("Scheduling retry %d/%d to poll test", retries, r.retries)
				r.afterInterval()
				continue
			}
			logPrintf("Aborting test after %d retries to poll test %s", r.retries, name)
			done <- i
			return
		}
		retries = 0
		config.Status = loadTest.Status
		s = status
		status = statusString(config)
		switch {
		case loadTest.Status.State.IsTerminated():
			logPrintf("%s", status)
			done <- i
			return
		case loadTest.Status.State == grpcv1.Running:
			logPrintf("%s", status)
			r.afterInterval()
		default:
			if s != status {
				logPrintf("%s", status)
			}
			// Use a longer polling interval for tests that have not started.
			r.afterInterval()
			r.afterInterval()
		}
	}
}

// nameString returns a string to represent the test name in logs.
// This string consists of two names: (1) the test name in the LoadTest
// metadata, (2) a test name derived from the prefix, scenario and uniquifier
// (if these elements are present in labels and annotations). This is a
// workaround for the fact that we cannot use the second name in the metadata.
// The LoadTest name is currently used as a label in pods, to refer back to the
// correspondingLoadTest (instead of the LoadTest UID). Labels are limited to
// 63 characters, while names themselves can go up to 253.
func nameString(config *grpcv1.LoadTest) string {
	var prefix, scenario string
	var ok bool
	if prefix, ok = config.Labels["prefix"]; !ok {
		return config.Name
	}
	if scenario, ok = config.Annotations["scenario"]; !ok {
		return config.Name
	}
	elems := []string{prefix}
	if scenario != "" {
		elems = append(elems, strings.Split(scenario, "_")...)
	}
	if uniquifier := config.Annotations["uniquifier"]; uniquifier != "" {
		elems = append(elems, uniquifier)
	}
	name := strings.Join(elems, "-")
	if name == config.Name {
		return config.Name
	}
	return fmt.Sprintf("%s [%s]", name, config.Name)
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
