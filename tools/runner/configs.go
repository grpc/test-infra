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
	"bufio"
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// DecodeFromFiles reads LoadTest configurations from a set of files.
// Each file is a multipart YAML file containing LoadTest configurations.
func DecodeFromFiles(fileNames []string) ([]*grpcv1.LoadTest, error) {
	var configs []*grpcv1.LoadTest
	for _, fileName := range fileNames {
		c, err := decodeFromFile(fileName)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c...)
	}
	return configs, nil
}

// decodeFromFile reads LoadTest configurations from a single file.
func decodeFromFile(fileName string) ([]*grpcv1.LoadTest, error) {
	var configs []*grpcv1.LoadTest
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(f)
	for {

		config, err := decodeNext(scanner)
		if err != nil {
			return nil, fmt.Errorf("error decoding config from %q: %v", fileName, err)
		}
		if config == nil {
			break
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// decodeNext decodes the next LoadTest configuration found in the file.
func decodeNext(scanner *bufio.Scanner) (*grpcv1.LoadTest, error) {
	const sep = "---"
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == sep {
			break
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return nil, nil
	}
	config := new(grpcv1.LoadTest)
	err := yaml.Unmarshal([]byte(strings.Join(lines, "\n")), config)
	return config, err
}
