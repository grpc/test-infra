/*
Copyright 2020 gRPC authors.

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

package podbuilder

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"

	"github.com/grpc/test-infra/config"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var stop chan struct{}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"PodBuilder Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var pools = map[string]int{
	"drivers":   3,
	"workers-a": 5,
	"workers-b": 7,
}

var nodes = func() []*corev1.Node {
	var items []*corev1.Node

	for pool, count := range pools {
		for i := 0; i < count; i++ {
			items = append(items, &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("node-%s-%d", pool, i),
					Labels: map[string]string{
						"pool": pool,
					},
				},
			})
		}
	}

	return items
}()

func newDefaults() *config.Defaults {
	return &config.Defaults{
		DriverPool:  "drivers",
		WorkerPool:  "workers-8core",
		CloneImage:  "gcr.io/grpc-fake-project/test-infra/clone",
		ReadyImage:  "gcr.io/grpc-fake-project/test-infra/ready",
		DriverImage: "gcr.io/grpc-fake-project/test-infra/driver",
		Languages: []config.LanguageDefault{
			{
				Language:   "cxx",
				BuildImage: "l.gcr.io/google/bazel:latest",
				RunImage:   "gcr.io/grpc-fake-project/test-infra/cxx",
			},
			{
				Language:   "go",
				BuildImage: "golang:1.14",
				RunImage:   "gcr.io/grpc-fake-project/test-infra/go",
			},
			{
				Language:   "java",
				BuildImage: "java:jdk8",
				RunImage:   "gcr.io/grpc-fake-project/test-infra/java",
			},
		},
	}
}
