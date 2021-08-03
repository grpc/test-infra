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

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/grpc/test-infra/tools/runner"
	"github.com/grpc/test-infra/tools/runner/xunit"
)

func main() {
	var i runner.FileNames
	var o string
	var xunitSuitesName string
	var c runner.ConcurrencyLevels
	var a string
	var p time.Duration
	var retries uint

	flag.Var(&i, "i", "input files containing load test configurations")
	flag.StringVar(&o, "o", "", "name of the output file for xunit xml report")
	flag.StringVar(&xunitSuitesName, "xunit-suites-name", "", "name field for testsuites in xunit xml report")
	flag.Var(&c, "c", "concurrency level, in the form [<queue name>:]<concurrency level>")
	flag.StringVar(&a, "annotation-key", "pool", "annotation key to parse for queue assignment")
	flag.DurationVar(&p, "polling-interval", 20*time.Second, "polling interval for load test status")
	flag.UintVar(&retries, "polling-retries", 2, "Maximum retries in case of communication failure")
	flag.Parse()

	inputConfigs, err := runner.DecodeFromFiles(i)
	if err != nil {
		log.Fatalf("Failed to decode: %v", err)
	}

	configQueueMap := runner.CreateQueueMap(inputConfigs, runner.QueueSelectorFromAnnotation(a))
	err = runner.ValidateConcurrencyLevels(configQueueMap, c)
	if err != nil {
		log.Fatalf("Failed to validate concurrency levels: %v", err)
	}

	log.Printf("Annotation key for queue assignment: %s", a)
	log.Printf("Polling interval: %v", p)
	log.Printf("Polling retries: %d", retries)
	log.Printf("Test counts per queue: %v", runner.CountConfigs(configQueueMap))
	log.Printf("Queue concurrency levels: %v", c)

	r := runner.NewRunner(runner.NewLoadTestGetter(), runner.AfterIntervalFunction(p), retries)

	logPrefixFmt := runner.LogPrefixFmt(configQueueMap)

	var report *xunit.Report
	if o != "" {
		report = &xunit.Report{
			Name: xunitSuitesName,
		}
	}

	reporter := runner.NewReporter(report)
	reporter.SetStartTime(time.Now())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan *runner.TestSuiteReporter)

	for qName, configs := range configQueueMap {
		testSuiteReporter := reporter.NewTestSuiteReporter(qName, logPrefixFmt)
		testSuiteReporter.SetStartTime(time.Now())
		go r.Run(ctx, configs, testSuiteReporter, c[qName], done)
	}

	for range configQueueMap {
		testSuiteReporter := <-done
		testSuiteReporter.SetEndTime(time.Now())
		log.Printf("Done running tests for queue %q in %s", testSuiteReporter.Queue(), testSuiteReporter.Duration())
	}

	reporter.SetEndTime(time.Now())

	if report != nil {
		report.Finalize()

		outputFile, err := os.Create(o)
		if err != nil {
			log.Fatalf("Failed to create output file %q: %v", o, err)
		}

		err = report.WriteToStream(outputFile, xunit.ReportWritingOptions{
			IndentSize: 2,
			MaxRetries: 3,
		})
		if err != nil {
			log.Fatalf("Failed to write XML report to output file %q: %v", o, err)
		}
	}
}
