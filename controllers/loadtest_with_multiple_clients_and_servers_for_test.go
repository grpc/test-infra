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

package controllers

import (
	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/pkg/defaults"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// +kubebuilder:scaffold:imports
)

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
	workerPool := "workers-8core"

	clientComponentName := "client-"
	serverComponentName := "server-"
	driverComponentName := "driver-1"

	//Create load test with 1 driver
	createdLoadTest := &grpcv1.LoadTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-loadtest-multiple-clients-and-servers",
			Namespace: "default",
		},

		Spec: grpcv1.LoadTestSpec{
			Driver: &grpcv1.Driver{
				Component: grpcv1.Component{
					Name:     &driverComponentName,
					Language: "cxx",
					Pool:     &driverPool,
					Run: grpcv1.Run{
						Image: &driverImage,
					},
				},
			},
			Results: &grpcv1.Results{
				BigQueryTable: &bigQueryTable,
			},

			Scenarios: []grpcv1.Scenario{
				{Name: "cpp-example-scenario"},
			},
		},
	}

	for i := 1; i <= 3; i++ {
		name := serverComponentName + string(i)
		createdLoadTest.Spec.Servers = append(createdLoadTest.Spec.Servers, grpcv1.Server{
			Component: grpcv1.Component{
				Name:     &name,
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
				Run: grpcv1.Run{
					Image:   &runImage,
					Command: runCommand,
					Args:    serverRunArgs,
				},
			},
		})
	}

	for i := 1; i <= 3; i++ {
		name := clientComponentName + string(i)
		createdLoadTest.Spec.Clients = append(createdLoadTest.Spec.Clients, grpcv1.Client{
			Component: grpcv1.Component{
				Name:     &name,
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
				Run: grpcv1.Run{
					Image:   &runImage,
					Command: runCommand,
					Args:    clientRunArgs,
				},
			},
		})
	}

	return createdLoadTest
}

func createPodListWithIrrelevantPod() *corev1.PodList {

	var currentPodList *corev1.PodList

	currentPodList.Items = append(currentPodList.Items,
		//add pods without metav1.ObjectMeta
		corev1.Pod{},

		//add pods with metav1.ObjectMeta but no labels are set
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random_name",
			},
		},

		//add pods with metav1.ObjectMeta but defaults.ComponentNameLabel labels are set
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random_name",
				Labels: map[string]string{
					defaults.LoadTestLabel: "random-task",
					defaults.RoleLabel:     "irrelevant role",
				},
			},
		},

		//add pods with metav1.ObjectMeta but defaults.ComponentNameLabel labels are set to random string
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random_name",
				Labels: map[string]string{
					defaults.LoadTestLabel:      "random-task",
					defaults.RoleLabel:          "irrelevant role",
					defaults.ComponentNameLabel: "irrelevant component name",
				},
			},
		},

		//correct loadtest name, wrong role, possible component name
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random_name",
				Labels: map[string]string{
					defaults.LoadTestLabel:      "test-loadtest-multiple-clients-and-servers",
					defaults.RoleLabel:          "driver",
					defaults.ComponentNameLabel: "server-1",
				},
			},
		},

		//correct loadtest name, wrong component name
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random_name",
				Labels: map[string]string{
					defaults.LoadTestLabel:      "test-loadtest-multiple-clients-and-servers",
					defaults.RoleLabel:          "server",
					defaults.ComponentNameLabel: "irrelevant component name",
				},
			},
		},
	)

	return currentPodList
}

func createPodListWithCurrentLoadTestPod(currentLoadTest *grpcv1.LoadTest, currentPodList *corev1.PodList) *corev1.PodList {
	//add all clients
	for _, eachClient := range currentLoadTest.Spec.Clients {
		currentPodList.Items = append(currentPodList.Items,
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "random_name",
					Labels: map[string]string{
						defaults.LoadTestLabel:      currentLoadTest.Name,
						defaults.RoleLabel:          "client",
						defaults.ComponentNameLabel: *eachClient.Name,
					},
				},
			})
	}
	//add all severs
	for _, eachServer := range currentLoadTest.Spec.Servers {
		currentPodList.Items = append(currentPodList.Items,
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "random_name",
					Labels: map[string]string{
						defaults.LoadTestLabel:      currentLoadTest.Name,
						defaults.RoleLabel:          "server",
						defaults.ComponentNameLabel: *eachServer.Name,
					},
				},
			})
	}

	currentPodList.Items = append(currentPodList.Items,
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random_name",
				Labels: map[string]string{
					defaults.LoadTestLabel:      currentLoadTest.Name,
					defaults.RoleLabel:          "Driver",
					defaults.ComponentNameLabel: *currentLoadTest.Spec.Driver.Name,
				},
			},
		})
	return currentPodList
}

//check if the two list is the same
func checkIfEqual(test []*grpcv1.Component, expected []*grpcv1.Component) bool {
	if len(test) != len(expected) {
		return false
	}
	for i := 0; i < len(test); i++ {
		if test[i].Name != expected[i].Name {
			return false
		}
	}
	return true
}
