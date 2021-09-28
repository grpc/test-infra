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
	"log"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// This side-effect import is required by GKE.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	clientset "github.com/grpc/test-infra/clientset"
)

// NewLoadTestGetter returns a client to interact with LoadTest resources.
// The client can be used to create, query for status and delete LoadTests.
func NewLoadTestGetter() clientset.LoadTestGetter {
	schemebuilder := runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(grpcv1.GroupVersion,
			&grpcv1.LoadTest{},
			&grpcv1.LoadTestList{},
		)
		metav1.AddToGroupVersion(scheme, grpcv1.GroupVersion)
		return nil
	})

	config := getConfig()

	schemebuilder.AddToScheme(clientgoscheme.Scheme)
	scheme := clientgoscheme.Scheme
	types := scheme.AllKnownTypes()
	_ = types

	grpcClientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create a grpc clientset: %v", err)
	}
	return grpcClientset.LoadTestV1().LoadTests(corev1.NamespaceDefault)
}

// getGenericClientset returns the standard K8 clientset.
func getGenericClientset() *kubernetes.Clientset {
	config := getConfig()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Could not create generic clientset")
	}

	return clientset
}

func getConfig() *rest.Config {
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
