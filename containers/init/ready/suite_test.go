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

package main

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// timeMultiplier provides a way to increase or decrease the timeouts for each
// test.
const timeMultiplier = 1

func TestReady(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ready Suite")
}

func newTestPod(role string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: role,
			Labels: map[string]string{
				"loadtest-role": role,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: types.UID("matching-test-uid"),
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: config.RunContainerName,
					Ports: []corev1.ContainerPort{
						{
							Name:          "driver",
							Protocol:      corev1.ProtocolTCP,
							ContainerPort: DefaultDriverPort,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			PodIP: "127.0.0.1",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
				},
			},
		},
	}
}

// newLoadTestWithMultipleClientsAndServers attempts to create a
// loadtest with multiple clients and servers for test.
func newLoadTestWithMultipleClientsAndServers(clientNum int, serverNum int) *grpcv1.LoadTest {
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

	serverNames := []string{}
	for i := 1; i <= serverNum; i++ {
		serverNames = append(serverNames, fmt.Sprintf("server-%d", i))
	}
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

	clientName := []string{}
	for i := 1; i <= clientNum; i++ {
		clientName = append(clientName, fmt.Sprintf("client-%d", i))
	}
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
	createdLoadTest.SetUID(types.UID("matching-test-uid"))
	return createdLoadTest
}
