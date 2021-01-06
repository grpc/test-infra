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
	"time"

	"github.com/google/uuid"
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
	"github.com/grpc/test-infra/optional"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var stop chan struct{}

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

func newLoadTest() *grpcv1.LoadTest {
	return &grpcv1.LoadTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid.New().String(),
			Namespace: corev1.NamespaceDefault,
		},
		Spec: grpcv1.LoadTestSpec{
			TimeoutSeconds: 300,
			TTLSeconds:     600,
			Driver: &grpcv1.Driver{
				Name:     optional.StringPtr("driver"),
				Language: "cxx",
				Pool:     optional.StringPtr("test-pool"),
				Run: grpcv1.Run{
					Image: optional.StringPtr("gcr.io/grpc-test-example/driver:v1"),
				},
			},
			Servers: []grpcv1.Server{
				{
					Name:     optional.StringPtr("server-1"),
					Language: "go",
					Pool:     optional.StringPtr("test-pool"),
					Clone: &grpcv1.Clone{
						Image:  optional.StringPtr("gcr.io/grpc-test-example/clone:v1"),
						Repo:   optional.StringPtr("https://github.com/grpc/test-infra.git"),
						GitRef: optional.StringPtr("master"),
					},
					Build: &grpcv1.Build{
						Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
						Command: []string{"go"},
						Args:    []string{"build", "-o", "server", "./server/main.go"},
					},
					Run: grpcv1.Run{
						Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
						Command: []string{"./server"},
						Args:    []string{"-verbose"},
					},
				},
			},
			Clients: []grpcv1.Client{
				{
					Name:     optional.StringPtr("client-1"),
					Language: "go",
					Pool:     optional.StringPtr("test-pool"),
					Clone: &grpcv1.Clone{
						Image:  optional.StringPtr("gcr.io/grpc-test-example/clone:v1"),
						Repo:   optional.StringPtr("https://github.com/grpc/test-infra.git"),
						GitRef: optional.StringPtr("master"),
					},
					Build: &grpcv1.Build{
						Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
						Command: []string{"go"},
						Args:    []string{"build", "-o", "client", "./client/main.go"},
					},
					Run: grpcv1.Run{
						Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					},
				},
			},
			Results: &grpcv1.Results{
				BigQueryTable: optional.StringPtr("example-dataset.example-table"),
			},
			ScenariosJSON: "{\"scenarios\": []}",
		},
		Status: grpcv1.LoadTestStatus{},
	}
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("setting gomega default timeouts")
	SetDefaultEventuallyTimeout(1500 * time.Millisecond)
	SetDefaultConsistentlyDuration(200 * time.Millisecond)

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
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Log:      ctrl.Log.WithName("controller").WithName("LoadTest"),
		Defaults: newDefaults(),
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
