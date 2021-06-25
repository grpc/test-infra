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
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// This side-effect import is required by GKE.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/google/uuid"
	grpcv1 "github.com/grpc/test-infra/api/v1"
	clientset "github.com/grpc/test-infra/clientset"
	"github.com/grpc/test-infra/optional"
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(grpcv1.GroupVersion,
			&grpcv1.LoadTest{},
			&grpcv1.LoadTestList{},
		)

		metav1.AddToGroupVersion(scheme, grpcv1.GroupVersion)
		return nil
	})
	addToScheme = schemeBuilder.AddToScheme
)

func main() {
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

	addToScheme(clientgoscheme.Scheme)
	scheme := clientgoscheme.Scheme
	types := scheme.AllKnownTypes()
	_ = types

	grpcClientset, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create a grpc clientset: %v", err)
	}
	testGetter := grpcClientset.LoadTestV1().LoadTests(corev1.NamespaceDefault)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	test, err := testGetter.Create(
		ctx,
		&grpcv1.LoadTest{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("client-test-manual-go-example-%s", uuid.New().String()),
				Labels: map[string]string{
					"Language": "go",
				},
			},
			Spec: grpcv1.LoadTestSpec{
				Servers: []grpcv1.Server{
					{
						Language: "go",
						Clone: &grpcv1.Clone{
							Repo:   optional.StringPtr("https://github.com/grpc/grpc-go.git"),
							GitRef: optional.StringPtr("master"),
						},
						Build: &grpcv1.Build{
							Command: []string{"go"},
							Args:    []string{"build", "-o", "/src/workspace/bin/worker", "./benchmark/worker"},
						},
						Run: grpcv1.Run{
							Command: []string{"/src/workspace/bin/worker"},
						},
					},
				},
				Clients: []grpcv1.Client{
					{
						Language: "go",
						Clone: &grpcv1.Clone{
							Repo:   optional.StringPtr("https://github.com/grpc/grpc-go.git"),
							GitRef: optional.StringPtr("master"),
						},
						Build: &grpcv1.Build{
							Command: []string{"go"},
							Args:    []string{"build", "-o", "/src/workspace/bin/worker", "./benchmark/worker"},
						},
						Run: grpcv1.Run{
							Command: []string{"/src/workspace/bin/worker"},
						},
					},
				},
				TimeoutSeconds: 900,
				TTLSeconds:     86400,
				ScenariosJSON: `
{
	"scenarios": [
		{
			"name": "go_generic_sync_streaming_ping_pong_secure",
			"warmup_seconds": 5,
			"benchmark_seconds": 30,
			"num_servers": 1,
			"server_config": {
				"async_server_threads": 1,
				"channel_args": [
					{
						"str_value": "latency",
						"name": "grpc.optimization_target"
					}
				],
				"payload_config": {
					"bytebuf_params": {
						"resp_size": 0,
						"req_size": 0
					}
				},
				"security_params": {
					"server_host_override": "foo.test.google.fr",
					"use_test_ca": true
				},
				"server_processes": 0,
				"server_type": "ASYNC_GENERIC_SERVER",
				"threads_per_cq": 0
			},
			"num_clients": 1,
			"client_config": {
				"async_client_threads": 1,
				"channel_args": [
					{
						"name": "grpc.optimization_target",
						"str_value": "latency"
					}
				],
				"client_channels": 1,
				"client_processes": 0,
				"client_type": "SYNC_CLIENT",
				"histogram_params": {
					"max_possible": 60000000000,
					"resolution": 0.01
				},
				"load_params": {
					"closed_loop": {}
				},
				"outstanding_rpcs_per_channel": 1,
				"payload_config": {
					"bytebuf_params": {
						"req_size": 0,
						"resp_size": 0
					}
				},
				"rpc_type": "STREAMING",
				"security_params": {
					"server_host_override": "foo.test.google.fr",
					"use_test_ca": true
				},
				"threads_per_cq": 0
			},
		}
	]
}`,
			},
		},
		metav1.CreateOptions{},
	)

	_, err = testGetter.Get(ctx, test.Name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to fetch the newly created test: %v", err)
	}

	testList, err := testGetter.List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("failed to list tests using client: %v", err)
	}
	found := false
	for _, t := range testList.Items {
		if t.Name == test.Name {
			found = true
			break
		}
	}
	if !found {
		log.Fatalf("failed to find newly created test in list: %v", err)
	}

	err = testGetter.Delete(ctx, test.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("failed to delete newly-created test using client: %v", err)
	}
}
