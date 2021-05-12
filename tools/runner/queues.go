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
	"errors"
	"fmt"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// QueueSelectorFunction maps a LoadTest configuration to an execution queue.
type QueueSelectorFunction = func(*grpcv1.LoadTest) string

// QueueSelectorFromAnnotation sets up key selection from a config annotation.
// This function returns a queue selector function that looks for a specific
// key annotation and returns the value of the annotation.
func QueueSelectorFromAnnotation(key string) QueueSelectorFunction {
	return func(config *grpcv1.LoadTest) string {
		return config.Annotations[key]
	}
}

// CreateQueueMap maps LoadTest configurations into execution queues.
// Configurations are mapped into queues using a queue selector.
func CreateQueueMap(configs []*grpcv1.LoadTest, qs QueueSelectorFunction) map[string][]*grpcv1.LoadTest {
	m := make(map[string][]*grpcv1.LoadTest)
	for _, config := range configs {
		qName := qs(config)
		m[qName] = append(m[qName], config)
	}
	return m
}

// ValidateConcurrencyLevels checks that all queues have levels defined.
// LoadTests are mapped into queues and run concurrently. A concurrency level
// must be specified for each queue.
func ValidateConcurrencyLevels(configMap map[string][]*grpcv1.LoadTest, concurrencyLevels map[string]int) error {
	for qName := range configMap {
		if _, ok := concurrencyLevels[qName]; !ok {
			if qName != "" {
				return fmt.Errorf("no concurrency level specified for queue %q", qName)
			}
			return errors.New("no concurrency level specified for global queue")
		}
	}
	return nil
}

// CountConfigs counts the number of configs in each queue.
func CountConfigs(configMap map[string][]*grpcv1.LoadTest) map[string]int {
	m := make(map[string]int)
	for qName, configs := range configMap {
		m[qName] = len(configs)
	}
	return m
}
