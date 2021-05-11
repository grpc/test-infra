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
	Done           chan string
	loadTestGetter clientset.LoadTestGetter
	afterInterval  func()
	retries        uint
}

// NewRunner creates a new Runner object.
func NewRunner(loadTestGetter clientset.LoadTestGetter, afterInterval func(), retries uint) *Runner {
	return &Runner{
		Done:           make(chan string),
		loadTestGetter: loadTestGetter,
		afterInterval:  afterInterval,
		retries:        retries,
	}
}

// Run runs a set of LoadTests at a given concurrency level.
func (r *Runner) Run(qName string, configs []*grpcv1.LoadTest, concurrencyLevel int) {
	done := make(chan int)
	n := 0
	for i, config := range configs {
		if n < concurrencyLevel {
			go r.runTest(qName, config, i, done)
			n++
			continue
		}
		<-done
		n--
	}
	for n > 0 {
		<-done
		n--
	}
	r.Done <- qName
}

// runTest creates a single LoadTest and monitors it to completion.
func (r *Runner) runTest(qName string, config *grpcv1.LoadTest, i int, done chan int) {
	id := fmt.Sprintf("%-14s %3d", qName, i)
	name := nameString(config)
	var status string
	var retries uint

	for {
		loadTest, err := r.loadTestGetter.Create(config, metav1.CreateOptions{})
		if err != nil {
			log.Printf("[%s] Failed to create test %s", id, name)
			if retries < r.retries {
				retries++
				log.Printf("[%s] Scheduling retry %d/%d to create test", id, retries, r.retries)
				r.afterInterval()
				continue
			}
			log.Printf("[%s] Aborting after %d retries to create test %s", id, r.retries, name)
			done <- i
			return
		}
		retries = 0
		config.Status = loadTest.Status
		log.Printf("[%s] Created test %s", id, name)
		break
	}

	for {
		loadTest, err := r.loadTestGetter.Get(config.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("[%s] Failed to poll test %s", id, name)
			if retries < r.retries {
				retries++
				log.Printf("[%s] Scheduling retry %d/%d to poll test", id, retries, r.retries)
				r.afterInterval()
				continue
			}
			log.Printf("[%s] Aborting test after %d retries to poll test %s", id, r.retries, name)
			done <- i
			return
		}
		retries = 0
		config.Status = loadTest.Status
		s := status
		status := statusString(config)
		switch {
		case loadTest.Status.State.IsTerminated():
			log.Printf("[%s] %s", id, status)
			done <- i
			return
		case loadTest.Status.State == grpcv1.Running:
			log.Printf("[%s] %s", id, status)
			r.afterInterval()
		default:
			if s != status {
				log.Printf("[%s] %s", id, status)
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
