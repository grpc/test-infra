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
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	grpcv1 "github.com/grpc/test-infra/api/v1"
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
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = grpcv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: ":3777",
		Port:               9443,
	})
	Expect(err).ToNot(HaveOccurred())

	reconciler := &LoadTestReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Log:    ctrl.Log.WithName("controller").WithName("LoadTest"),
	}
	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	stop = make(chan struct{})
	go func() {
		err := k8sManager.Start(stop)
		Expect(err).ToNot(HaveOccurred())
	}()

	for _, node := range nodes {
		Expect(k8sClient.Create(context.Background(), node)).To(Succeed())
	}

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	close(stop)
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

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

func newLoadTest() *grpcv1.LoadTest {
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

	clientComponentName := "client-1"
	serverComponentName := "server-1"
	driverComponentName := "driver-1"

	return &grpcv1.LoadTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-loadtest",
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

			Servers: []grpcv1.Server{
				{
					Component: grpcv1.Component{
						Name:     &serverComponentName,
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
				},
			},

			Clients: []grpcv1.Client{
				{
					Component: grpcv1.Component{
						Name:     &clientComponentName,
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
	workerPool := "workers-8core"

	driverComponentName := "driver-1"

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

	serverNames := []string{"server-1", "server-2", "server-3"}
	for i := 1; i <= len(serverNames); i++ {
		createdLoadTest.Spec.Servers = append(createdLoadTest.Spec.Servers, grpcv1.Server{
			Component: grpcv1.Component{
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
				Run: grpcv1.Run{
					Image:   &runImage,
					Command: runCommand,
					Args:    serverRunArgs,
				},
			},
		})
	}

	clientName := []string{"client-1", "client-2", "client-3"}
	for i := 1; i <= len(clientName); i++ {
		createdLoadTest.Spec.Clients = append(createdLoadTest.Spec.Clients, grpcv1.Client{
			Component: grpcv1.Component{
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

// populatePodListWithIrrelevantPod attempt to create pod list and populate it with
// irrelevant pods
func createPodListWithIrrelevantPod() *corev1.PodList {
	currentPodList := &corev1.PodList{Items: []corev1.Pod{}}
	currentPodList.Items = append(currentPodList.Items,
		corev1.Pod{},

		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{},
		},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
			},
		},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
				Labels: map[string]string{
					"keyOne": "random-task",
					"KeyTwo": "irrelevant role",
				},
			},
		},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
				Labels: map[string]string{
					config.LoadTestLabel: "random-task",
					config.RoleLabel:     "irrelevant role",
				},
			},
		},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
				Labels: map[string]string{
					config.LoadTestLabel:      "random-task",
					config.RoleLabel:          "irrelevant role",
					config.ComponentNameLabel: "irrelevant component name",
				},
			},
		},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
				Labels: map[string]string{
					config.LoadTestLabel:      "test-loadtest-multiple-clients-and-servers",
					config.RoleLabel:          "driver",
					config.ComponentNameLabel: "server-1",
				},
			},
		},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
				Labels: map[string]string{
					config.LoadTestLabel:      "test-loadtest-multiple-clients-and-servers",
					config.RoleLabel:          "server",
					config.ComponentNameLabel: "irrelevant component name",
				},
			},
		},
	)

	return currentPodList
}

// populatePodListWithCurrentLoadTestPod attempts to create a Podlist and populate
// it with the pods came from current loadtest.
func populatePodListWithCurrentLoadTestPod(currentLoadTest *grpcv1.LoadTest) *corev1.PodList {
	currentPodList := &corev1.PodList{Items: []corev1.Pod{}}

	for _, eachClient := range currentLoadTest.Spec.Clients {
		currentPodList.Items = append(currentPodList.Items,
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "random-name",
					Labels: map[string]string{
						config.LoadTestLabel:      currentLoadTest.Name,
						config.RoleLabel:          "client",
						config.ComponentNameLabel: *eachClient.Name,
					},
				},
			})
	}

	for _, eachServer := range currentLoadTest.Spec.Servers {
		currentPodList.Items = append(currentPodList.Items,
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "random-name",
					Labels: map[string]string{
						config.LoadTestLabel:      currentLoadTest.Name,
						config.RoleLabel:          "server",
						config.ComponentNameLabel: *eachServer.Name,
					},
				},
			})
	}

	currentPodList.Items = append(currentPodList.Items,
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "random-name",
				Labels: map[string]string{
					config.LoadTestLabel:      currentLoadTest.Name,
					config.RoleLabel:          "driver",
					config.ComponentNameLabel: *currentLoadTest.Spec.Driver.Name,
				},
			},
		})

	return currentPodList
}
