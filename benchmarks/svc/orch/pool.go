// Copyright 2020 gRPC authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orch

import (
	"context"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pool describes a cluster of identical machines.
type Pool struct {
	// Name is an indentifier that uniquely distinguishes a pool instance.
	Name string

	// Available is the number of machines that are idle and able to be reserved.
	Available int

	// Capacity is the total number of machines in the pool.
	Capacity int
}

// PoolAdder is a type that can add and remove pools.
type PoolAdder interface {
	// AddPool adds a pool.
	AddPool(pool Pool)

	// RemovePool removes the pool.
	RemovePool(pool Pool)
}

// NodeLister is any type that can list nodes in a kubernetes namespace. Most likely, this will be
// the kubernetes type that implements the v1.NodeInterface.
type NodeLister interface {
	// List returns a list of Kubernetes nodes or an error.
	List(context.Context, metav1.ListOptions) (*v1.NodeList, error)
}

// FindPools uses a NodeLister to find all nodes, determine their pools and produce Pool instances
// with their capacities. It returns a map where the string is the name of the pool and the value is
// the Pool instance. An error is returned if the List operation on the NodeLister errors.
func FindPools(nl NodeLister) (map[string]Pool, error) {
	nodeList, err := nl.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodes := nodeList.Items
	pm := make(map[string]Pool)
	for _, node := range nodes {
		labels := node.ObjectMeta.Labels
		if labels == nil {
			continue
		}

		poolName, ok := labels["pool"]
		if !ok {
			continue
		}

		pool, ok := pm[poolName]
		if !ok {
			pool = Pool{Name: poolName}
		}

		pool.Available++
		pool.Capacity++
		pm[poolName] = pool
	}

	return pm, nil
}
