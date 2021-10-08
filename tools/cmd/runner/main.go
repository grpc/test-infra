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
	"path"
	"time"

	"github.com/grpc/test-infra/tools/runner"
	"github.com/grpc/test-infra/tools/runner/xunit"
)

func main() {
	var i runner.FileNames
	var o string
	var c runner.ConcurrencyLevels
	var a string
	var p time.Duration
	var retries uint
	var deleteSuccessfulTests bool

	flag.Var(&i, "i", "input files containing load test configurations")
	flag.StringVar(&o, "o", "", "name of the output file for xunit xml report")
	flag.Var(&c, "c", "concurrency level, in the form [<queue name>:]<concurrency level>")
	flag.StringVar(&a, "annotation-key", "pool", "annotation key to parse for queue assignment")
	flag.DurationVar(&p, "polling-interval", 20*time.Second, "polling interval for load test status")
	flag.UintVar(&retries, "polling-retries", 2, "Maximum retries in case of communication failure")
	flag.BoolVar(&deleteSuccessfulTests, "delete-successful-tests", false, "Delete tests immediately in case of successful termination")
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

	outputPath := xunit.OutputPath(o)

	outputDirMap := make(map[string]string)
	for qName := range configQueueMap {
		outputFilePath := outputPath(qName)
		outputDir := path.Dir(outputFilePath)
		if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
			log.Fatalf("Failed to create output directory %q: %v", outputDir, err)
		}
		outputDirMap[qName] = outputDir
	}

	log.Printf("Annotation key for queue assignment: %s", a)
	log.Printf("Polling interval: %v", p)
	log.Printf("Polling retries: %d", retries)
	log.Printf("Test counts per queue: %v", runner.CountConfigs(configQueueMap))
	log.Printf("Queue concurrency levels: %v", c)
	log.Printf("Output directories: %v", outputDirMap)

	r := runner.NewRunner(runner.NewLoadTestGetter(), runner.AfterIntervalFunction(p), retries, deleteSuccessfulTests)

	logPrefixFmt := runner.LogPrefixFmt(configQueueMap)

	report := xunit.Report{}

	reporter := runner.NewReporter(&report)
	reporter.SetStartTime(time.Now())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan *runner.TestSuiteReporter)

	for qName, configs := range configQueueMap {
		testSuiteReporter := reporter.NewTestSuiteReporter(qName, logPrefixFmt, runner.TestCaseNameFromAnnotations("scenario"))
		testSuiteReporter.SetStartTime(time.Now())
		go r.Run(ctx, configs, testSuiteReporter, c[qName], outputDirMap[qName], done)
	}

	for range configQueueMap {
		testSuiteReporter := <-done
		testSuiteReporter.SetEndTime(time.Now())
		log.Printf("Done running tests for queue %q in %s", testSuiteReporter.Queue(), testSuiteReporter.Duration())
	}

	reporter.SetEndTime(time.Now())

	report.Finalize()

	if o != "" {
		for suiteName, suiteReport := range report.Split() {
			outputFilePath := outputPath(suiteName)

			outputFile, err := os.Create(outputFilePath)
			if err != nil {
				log.Fatalf("Failed to create output file %q: %v", outputFilePath, err)
			}

			err = suiteReport.WriteToStream(outputFile, xunit.ReportWritingOptions{
				IndentSize: 2,
				MaxRetries: 3,
			})
			if err != nil {
				log.Fatalf("Failed to write XML report to file %q: %v", outputFilePath, err)
			}

			if err := outputFile.Close(); err != nil {
				log.Fatalf("Failed to close output file %q: %v", outputFilePath, err)
			}

			log.Printf("Wrote XML report to file %q", outputFilePath)
		}
	}

	if report.ErrorCount > 0 {
		log.Fatalf("Errors found during test run: %d", report.ErrorCount)
	}
}
