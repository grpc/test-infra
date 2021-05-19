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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/grpc/test-infra/tools/runner"
)

// defaultOutputSuiteName provides a default name for the testsuites tag
// in an XML report. It is based on the number of nanoseconds since the
// UNIX epoch.
var defaultOutputSuiteName = fmt.Sprintf("benchmarks-%d", time.Now().UnixNano())

func main() {
	var i runner.FileNames
	var o runner.FileNames
	var c runner.ConcurrencyLevels
	var a string
	var p time.Duration
	var retries uint
	var outputSuiteName string

	flag.Var(&i, "i", "input files containing load test configurations")
	flag.Var(&c, "c", "concurrency level, in the form [<queue name>:]<concurrency level>")
	flag.Var(&o, "o", "name of the output file for junit xml report")
	flag.StringVar(&a, "annotation-key", "pool", "annotation key to parse for queue assignment")
	flag.StringVar(&outputSuiteName, "output-suite-name", defaultOutputSuiteName, "name field for testsuites in junit xml report")
	flag.DurationVar(&p, "polling-interval", 20*time.Second, "polling interval for load test status")
	flag.UintVar(&retries, "polling-retries", 2, "Maximum retries in case of communication failure")
	flag.Parse()

	if len(o) > 1 {
		log.Fatalf("Only one output file can be written per run")
	}

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

	done := make(chan string)

	suitesReporter := runner.NewTestSuitesReporter(outputSuiteName)

	suitesReporter.SetStartTime(time.Now())

	for qName, configs := range configQueueMap {
		suiteReporter := suitesReporter.NewTestSuiteReporter(qName, logPrefixFmt)
		go r.Run(configs, suiteReporter, c[qName], done)
	}

	for range configQueueMap {
		qName := <-done
		log.Printf("Done running tests for queue %q", qName)
	}

	suitesReporter.SetEndTime(time.Now())

	if len(o) > 0 {
		xmlReport, err := suitesReporter.XMLReport()
		if err != nil {
			log.Fatalf("Failed to marshal xml report: %v", err)
		}

		err = ioutil.WriteFile(o[0], xmlReport, 0666)
		if err != nil {
			log.Printf("Failed to write output file: %v", err)
		}

		log.Printf("Wrote XML report (%d bytes) to file: %v", len(xmlReport), o[0])
	}
}
