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

package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	corev1 "k8s.io/api/core/v1"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// This side-effect import is required by GKE.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	clientset "github.com/grpc/test-infra/clientset"
	"github.com/grpc/test-infra/status"
)

// NewLoadTestGetter returns a client to interact with LoadTest resources. The
// client can be used to create, query for status and delete LoadTests.
func NewLoadTestGetter() clientset.LoadTestGetter {
	clientset := NewGRPCTestClientset()
	schemebuilder := runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(grpcv1.GroupVersion,
			&grpcv1.LoadTest{},
			&grpcv1.LoadTestList{},
		)
		metav1.AddToGroupVersion(scheme, grpcv1.GroupVersion)
		return nil
	})

	schemebuilder.AddToScheme(clientgoscheme.Scheme)
	scheme := clientgoscheme.Scheme
	types := scheme.AllKnownTypes()
	_ = types

	return clientset.LoadTestV1().LoadTests(corev1.NamespaceDefault)
}

// NewGRPCTestClientset returns a new GRPCTestClientset.
func NewGRPCTestClientset() clientset.GRPCTestClientset {
	config := getKubernetesConfig()
	grpcClientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create a grpc clientset: %v", err)
	}
	return grpcClientset
}

// NewK8sClientset returns a new Kubernetes clientset.
func NewK8sClientset() *kubernetes.Clientset {
	config := getKubernetesConfig()
	k8sClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create a k8 clientset: %v", err)
	}
	return k8sClientset
}

// NewPodsGetter returns a new PodsGetter.
func NewPodsGetter() corev1types.PodsGetter {
	clientset := NewK8sClientset()
	return clientset.CoreV1()
}

// GetTestPods retrieves the pods associated with a LoadTest.
func GetTestPods(ctx context.Context, loadTest *grpcv1.LoadTest, podsGetter corev1types.PodsGetter) ([]*corev1.Pod, error) {
	podLister := podsGetter.Pods(metav1.NamespaceAll)
	// Get a list of all pods
	podList, err := podLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch list of pods: %v", err)
	}

	// Get pods just for this specific test
	testPods := status.PodsForLoadTest(loadTest, podList.Items)
	return testPods, nil
}

// getKubernetesConfig retrieves the kubernetes configuration.
func getKubernetesConfig() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		if err != rest.ErrNotInCluster {
			log.Fatalf("failed to connect within cluster: %v", err)
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("could not find a home directory for user: %v", err)
		}

		cfgPathBuilder := &strings.Builder{}
		cfgPathBuilder.WriteString(homeDir)
		if homeDir[:len(homeDir)-1] != "/" {
			cfgPathBuilder.WriteString("/")
		}
		cfgPathBuilder.WriteString(".kube/config")
		cfgPath := cfgPathBuilder.String()

		config, err = clientcmd.BuildConfigFromFlags("", cfgPath)
		if err != nil {
			log.Fatalf("failed to construct config for path %q: %v", cfgPath, err)
		}
	}
	return config
}
