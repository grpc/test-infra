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
	"fmt"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	DefaultDriverPool = "drivers"
	DefaultWorkerPool = "workers-8core"
	LoadTestLabel     = "loadtest"
	RoleLabel         = "loadtest-role"
	ServerRole        = "server"
	ClientRole        = "client"
	DriverRole        = "driver"
)

type PoolManager struct {
	availability map[string]int
	capacity     map[string]int
}

func (pm *PoolManager) initialize() {
	if pm.availability == nil {
		pm.availability = make(map[string]int)
	}
	if pm.capacity == nil {
		pm.capacity = make(map[string]int)
	}
}

func (pm *PoolManager) Capacity(pool string) (int, error) {
	pm.initialize()
	capacity, ok := pm.capacity[pool]
	if !ok {
		return 0, fmt.Errorf("pool %q does not exist", pool)
	}
	return capacity, nil
}

func (pm *PoolManager) Exists(pool string) bool {
	pm.initialize()
	_, capacityOk := pm.capacity[pool]
	_, availabilityOk := pm.availability[pool]
	return capacityOk && availabilityOk
}

func (pm *PoolManager) AddNode(pool string) {
	if !pm.Exists(pool) {
		pm.capacity[pool] = 0
		pm.availability[pool] = 0
	}

	pm.capacity[pool]++
	pm.availability[pool]++
}

func (pm *PoolManager) AddNodeList(nodeList corev1.NodeList) {
	nodes := nodeList.Items
	for _, node := range nodes {
		labels := node.ObjectMeta.Labels
		if labels == nil {
			continue
		}

		pool, ok := labels["pool"]
		if !ok {
			continue
		}

		pm.AddNode(pool)
	}
}

func (pm *PoolManager) AvailableNodes(pool string) (int, error) {
	pm.initialize()
	available, ok := pm.availability[pool]
	if !ok {
		return 0, fmt.Errorf("pool %q does not exist", pool)
	}
	return available, nil
}

func (pm *PoolManager) setAvailableNodes(pool string, delta int) error {
	if !pm.Exists(pool) {
		return fmt.Errorf("pool %q does not exist", pool)
	}

	a := pm.availability[pool]
	c := pm.capacity[pool]
	da := a + delta

	if da < 0 {
		return fmt.Errorf("pool %q availability cannot drop below zero", pool)
	}
	if da > c {
		return fmt.Errorf("pool %q availability cannot exceed its capacity (%d nodes)", pool, c)
	}

	pm.availability[pool] = da
	return nil
}

func (pm *PoolManager) AvailableNodesDec(pool string) error {
	return pm.setAvailableNodes(pool, -1)
}

func (pm *PoolManager) AvailableNodesInc(pool string) error {
	return pm.setAvailableNodes(pool, 1)
}

func (pm *PoolManager) UpdateAvailability(loadTestList grpcv1.LoadTestList, pendingTestSet *LoadTestSet) error {
	loadTests := loadTestList.Items
	for _, loadTest := range loadTests {
		if !pendingTestSet.Includes(loadTest) {
			continue
		}

		components := getComponents(&loadTest)
		for _, component := range components {
			if err := pm.AvailableNodesDec(*component.Pool); err != nil {
				return err
			}
		}
	}
	return nil
}

func (pm *PoolManager) Fits(loadTest grpcv1.LoadTest) (bool, error) {
	pm.initialize()

	// build up a map of pool names to number of nodes the test requires
	requirements := make(map[string]int)

	components := getComponents(&loadTest)
	for _, component := range components {
		c, ok := requirements[*component.Pool]
		if !ok {
			requirements[*component.Pool] = 1
		} else {
			requirements[*component.Pool] = c + 1
		}
	}

	for pool, requiredNodes := range requirements {
		available, err := pm.AvailableNodes(pool)
		if err != nil {
			return false, err
		}

		if requiredNodes > available {
			return false, nil
		}
	}

	return true, nil
}

func getComponents(loadtest *grpcv1.LoadTest) []*grpcv1.Component {
	spec := &loadtest.Spec

	if spec.Driver.Pool == nil {
		spec.Driver.Pool = &DefaultDriverPool
	}

	components := []*grpcv1.Component{&spec.Driver.Component}

	for _, server := range spec.Servers {
		server.Pool = &DefaultWorkerPool
		components = append(components, &server.Component)
	}

	for _, client := range spec.Clients {
		client.Pool = &DefaultWorkerPool
		components = append(components, &client.Component)
	}

	return components
}
