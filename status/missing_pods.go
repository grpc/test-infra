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

// LoadTestMissing defines missing pods of LoadTest.
type LoadTestMissing struct {
	// Driver is the component that orchestrates the test. If Driver is not set
	// that means we already have the Driver running.
	Driver *grpcv1.Driver `json:"driver,omitempty"`

	// Servers are a list of components that receive traffic from. The list
	// indicates the Servers still in need.
	Servers []grpcv1.Server `json:"servers,omitempty"`

	// Clients are a list of components that send traffic to servers. The list
	// indicates the Clients still in need.
	Clients []grpcv1.Client `json:"clients,omitempty"`
}

// CheckMissingPods attempts to check if any required component is missing from
// the current load test. It takes reference of the current load test and a pod
// list that contains all running pods at the moment, returning all missing
// components required from the current load test with their roles.
func CheckMissingPods(currentLoadTest *grpcv1.LoadTest, ownedPods []*corev1.Pod) *LoadTestMissing {

	currentMissing := &LoadTestMissing{Servers: []grpcv1.Server{}, Clients: []grpcv1.Client{}}

	requiredClientMap := make(map[string]*grpcv1.Client)
	requiredServerMap := make(map[string]*grpcv1.Server)
	foundDriver := false

	for i := 0; i < len(currentLoadTest.Spec.Clients); i++ {
		requiredClientMap[*currentLoadTest.Spec.Clients[i].Name] = &currentLoadTest.Spec.Clients[i]
	}
	for i := 0; i < len(currentLoadTest.Spec.Servers); i++ {
		requiredServerMap[*currentLoadTest.Spec.Servers[i].Name] = &currentLoadTest.Spec.Servers[i]
	}

	if ownedPods != nil {

		for _, eachPod := range ownedPods {

			if eachPod.Labels == nil {
				continue
			}

			roleLabel := eachPod.Labels[config.RoleLabel]
			componentNameLabel := eachPod.Labels[config.ComponentNameLabel]

			if roleLabel == config.DriverRole {
				if *currentLoadTest.Spec.Driver.Component.Name == componentNameLabel {
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

	for _, eachMissingClient := range requiredClientMap {
		currentMissing.Clients = append(currentMissing.Clients, *eachMissingClient)
	}

	for _, eachMissingServer := range requiredServerMap {
		currentMissing.Servers = append(currentMissing.Servers, *eachMissingServer)
	}

	if !foundDriver {
		currentMissing.Driver = currentLoadTest.Spec.Driver
	}

	return currentMissing
}
