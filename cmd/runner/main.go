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
	"log"
	"time"

	"github.com/grpc/test-infra/tools/runner"
)

func main() {
	i := runner.FileNames(make([]string, 0))
	c := runner.ConcurrencyLevels(make(map[string]int))
	var a string
	var p time.Duration
	var retries uint

	flag.Var(&i, "i", "input files containing load test configurations")
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

	for qName, configs := range configQueueMap {
		go r.Run(qName, configs, c[qName])
	}

	for range configQueueMap {
		qName := <-r.Done
		log.Printf("Done running tests for queue %q", qName)
	}
}
