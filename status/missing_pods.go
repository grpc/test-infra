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
	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	corev1 "k8s.io/api/core/v1"
)

// DefaultClientPool is a key in the NodeCountByPool map on the LoadTestMissing
// struct. It maps to the number of nodes required from the default client pool.
const DefaultClientPool = "__default_pool (clients)"

// DefaultDriverPool is a key in the NodeCountByPool map on the LoadTestMissing
// struct. It maps to the number of nodes required from the default driver pool.
const DefaultDriverPool = "__default_pool (drivers)"

// DefaultServerPool is a key in the NodeCountByPool map on the LoadTestMissing
// struct. It maps to the number of nodes required from the default server pool.
const DefaultServerPool = "__default_pool (servers)"

// LoadTestMissing defines missing pods of LoadTest.
type LoadTestMissing struct {
	// Driver is the component that orchestrates the test. If Driver is not set
	// that means we already have the Driver running.
	Driver *grpcv1.Driver

	// Servers are a list of components that receive traffic from. The list
	// indicates the Servers still in need.
	Servers []grpcv1.Server

	// Clients are a list of components that send traffic to servers. The list
	// indicates the Clients still in need.
	Clients []grpcv1.Client

	// NodeCountPyPool is a map which gives the number of nodes required from
	// each pool to run the test. These counts will not include any pods from
	// the test that have already been scheduled. The names of the required node
	// pools are the keys and the value is the number of nodes required from the
	// named pool.
	//
	// If a test does not require any nodes from a pool, the pool name will not
	// be present as a key in the map. If a driver, client or server has not
	// been scheduled for a test and does not specify a pool, it will be counted
	// in one of the default pool keys. See the DefaultClientPool,
	// DefaultDriverPool and DefaultServerPool constants.
	NodeCountByPool map[string]int
}

// IsEmpty returns true if there are no missing driver, servers or clients on a
// LoadTestMissing struct. Otherwise, it returns false.
func (ltm *LoadTestMissing) IsEmpty() bool {
	return ltm.Driver == nil && len(ltm.Servers) == 0 && len(ltm.Clients) == 0
}

// CheckMissingPods attempts to check if any required component is missing from
// the current load test. It takes reference of the current load test and a pod
// list that contains all running pods at the moment, returning all missing
// components required from the current load test with their roles.
func CheckMissingPods(test *grpcv1.LoadTest, ownedPods []*corev1.Pod) *LoadTestMissing {
	currentMissing := &LoadTestMissing{
		Servers: []grpcv1.Server{},
		Clients: []grpcv1.Client{},
		NodeCountByPool: map[string]int{
			DefaultClientPool: 0,
			DefaultDriverPool: 0,
			DefaultServerPool: 0,
		},
	}

	requiredClientMap := make(map[string]*grpcv1.Client)
	requiredServerMap := make(map[string]*grpcv1.Server)
	foundDriver := false

	for i := 0; i < len(test.Spec.Clients); i++ {
		requiredClientMap[*test.Spec.Clients[i].Name] = &test.Spec.Clients[i]
	}
	for i := 0; i < len(test.Spec.Servers); i++ {
		requiredServerMap[*test.Spec.Servers[i].Name] = &test.Spec.Servers[i]
	}

	if ownedPods != nil {

		for _, eachPod := range ownedPods {

			if eachPod.Labels == nil {
				continue
			}

			roleLabel := eachPod.Labels[config.RoleLabel]
			componentNameLabel := eachPod.Labels[config.ComponentNameLabel]

			if roleLabel == config.DriverRole {
				if *test.Spec.Driver.Name == componentNameLabel {
					foundDriver = true
				}
			} else if roleLabel == config.ClientRole {
				if _, ok := requiredClientMap[componentNameLabel]; ok {
					delete(requiredClientMap, componentNameLabel)
				}
			} else if roleLabel == config.ServerRole {
				if _, ok := requiredServerMap[componentNameLabel]; ok {
					delete(requiredServerMap, componentNameLabel)
				}
			}
		}
	}

	incNodeCount := func(pool string) {
		if _, ok := currentMissing.NodeCountByPool[pool]; !ok {
			currentMissing.NodeCountByPool[pool] = 0
		}
		currentMissing.NodeCountByPool[pool]++
	}

	for _, eachMissingClient := range requiredClientMap {
		currentMissing.Clients = append(currentMissing.Clients, *eachMissingClient)
		if eachMissingClient.Pool == nil {
			currentMissing.NodeCountByPool[DefaultClientPool]++
		} else {
			incNodeCount(*eachMissingClient.Pool)
		}
	}

	for _, eachMissingServer := range requiredServerMap {
		currentMissing.Servers = append(currentMissing.Servers, *eachMissingServer)
		if eachMissingServer.Pool == nil {
			currentMissing.NodeCountByPool[DefaultServerPool]++
		} else {
			incNodeCount(*eachMissingServer.Pool)
		}
	}

	if !foundDriver {
		currentMissing.Driver = test.Spec.Driver
		if test.Spec.Driver.Pool == nil {
			currentMissing.NodeCountByPool[DefaultDriverPool]++
		} else {
			incNodeCount(*test.Spec.Driver.Pool)
		}
	}

	return currentMissing
}
