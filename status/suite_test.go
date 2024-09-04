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

package status

import (
	"testing"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStatus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Status Suite")
}

// newLoadTestWithMultipleClientsAndServers attempts to create a
// loadtest with multiple clients and servers for test.
func newLoadTestWithMultipleClientsAndServers() *grpcv1.LoadTest {
	cloneImage := "docker.pkg.github.com/grpc/test-infra/clone"
	cloneRepo := "https://github.com/grpc/grpc.git"
	cloneGitRef := "master"

	buildImage := "l.gcr.io/google/bazel:latest"
	buildCommand := []string{"bazel"}
	buildArgs := []string{"build", "//test/cpp/qps:qps_worker"}

	driverImage := "docker.pkg.github.com/grpc/test-infra/driver"
	runImage := "docker.pkg.github.com/grpc/test-infra/cxx"
	runCommand := []string{"bazel-bin/test/cpp/qps/qps_worker"}

	clientRunArgs := []string{"--driver_port=10000"}
	serverRunArgs := append(clientRunArgs, "--server_port=10010")

	bigQueryTable := "grpc-testing.e2e_benchmark.foobarbuzz"

	driverPool := "drivers"
	workerPool := "workers"

	driverComponentName := "driver-1"

	createdLoadTest := &grpcv1.LoadTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-loadtest-multiple-clients-and-servers",
			Namespace: "default",
		},

		Spec: grpcv1.LoadTestSpec{
			Driver: &grpcv1.Driver{
				Name:     &driverComponentName,
				Language: "cxx",
				Pool:     &driverPool,
				Run: []corev1.Container{{
					Image: driverImage,
				}},
			},
			Results: &grpcv1.Results{
				BigQueryTable: &bigQueryTable,
			},

			ScenariosJSON: "{\"scenarios\": []}",
		},
	}

	serverNames := []string{"server-1", "server-2", "server-3"}
	for i := 1; i <= len(serverNames); i++ {
		createdLoadTest.Spec.Servers = append(createdLoadTest.Spec.Servers, grpcv1.Server{
			Name:     &serverNames[i-1],
			Language: "cxx",
			Pool:     &workerPool,
			Clone: &grpcv1.Clone{
				Image:  &cloneImage,
				Repo:   &cloneRepo,
				GitRef: &cloneGitRef,
			},
			Build: &grpcv1.Build{
				Image:   &buildImage,
				Command: buildCommand,
				Args:    buildArgs,
			},
			Run: []corev1.Container{{
				Image:   runImage,
				Command: runCommand,
				Args:    serverRunArgs,
			}},
		})
	}

	clientName := []string{"client-1", "client-2", "client-3"}
	for i := 1; i <= len(clientName); i++ {
		createdLoadTest.Spec.Clients = append(createdLoadTest.Spec.Clients, grpcv1.Client{
			Name:     &clientName[i-1],
			Language: "cxx",
			Pool:     &workerPool,
			Clone: &grpcv1.Clone{
				Image:  &cloneImage,
				Repo:   &cloneRepo,
				GitRef: &cloneGitRef,
			},
			Build: &grpcv1.Build{
				Image:   &buildImage,
				Command: buildCommand,
				Args:    buildArgs,
			},
			Run: []corev1.Container{{
				Image:   runImage,
				Command: runCommand,
				Args:    clientRunArgs,
			}},
		})
	}

	return createdLoadTest
}

// populatePodListWithCurrentLoadTestPod attempts to create a Podlist and populate
// it with the pods came from current loadtest.
func populatePodListWithCurrentLoadTestPod(currentLoadTest *grpcv1.LoadTest) []*corev1.Pod {
	var currentPodList []*corev1.Pod

	for _, eachClient := range currentLoadTest.Spec.Clients {
		currentPodList = append(currentPodList,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "random-name",
					Labels: map[string]string{
						config.RoleLabel:          "client",
						config.ComponentNameLabel: *eachClient.Name,
					},
				},
			})
	}

	for _, eachServer := range currentLoadTest.Spec.Servers {
		currentPodList = append(currentPodList,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "random-name",
					Labels: map[string]string{
						config.RoleLabel:          "server",
						config.ComponentNameLabel: *eachServer.Name,
					},
				},
			})
	}

	currentPodList = append(currentPodList,
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
				Labels: map[string]string{
					config.RoleLabel:          "driver",
					config.ComponentNameLabel: *currentLoadTest.Spec.Driver.Name,
				},
			},
		})

	return currentPodList
}
