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
	"errors"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindPools(t *testing.T) {
	cases := []struct {
		description string
		nodes       []corev1.Node
		pools       map[string]Pool
	}{
		{
			description: "all nodes have pool label",
			nodes: []corev1.Node{
				newNodeInPool(t, "pool-one"),

				newNodeInPool(t, "pool-two"),
				newNodeInPool(t, "pool-two"),
				newNodeInPool(t, "pool-two"),

				newNodeInPool(t, "pool-three"),
				newNodeInPool(t, "pool-three"),
				newNodeInPool(t, "pool-three"),
			},
			pools: map[string]Pool{
				"pool-one":   {Name: "pool-one", Available: 1, Capacity: 1},
				"pool-two":   {Name: "pool-two", Available: 3, Capacity: 3},
				"pool-three": {Name: "pool-three", Available: 3, Capacity: 3},
			},
		},
		{
			description: "nodes with missing labels ignored",
			nodes: []corev1.Node{
				newNodeInPool(t, "pool-one"),

				// node without any labels
				{},

				// node without pool label
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"unrelated-label": "true",
						},
					},
				},
			},
			pools: map[string]Pool{
				"pool-one": {Name: "pool-one", Available: 1, Capacity: 1},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			nlm := &nodeListerMock{nodes: tc.nodes}

			pools, err := FindPools(nlm)
			if err != nil {
				t.Fatalf("unexpected error in find pools using node lister: %v", err)
			}

			if !reflect.DeepEqual(tc.pools, pools) {
				t.Fatalf("expected pool mismatch, wanted %v but got %v", tc.pools, pools)
			}
		})
	}

	t.Run("kubernetes issue returns error", func(t *testing.T) {
		nlm := &nodeListerMock{}
		nlm.err = errors.New("fake kubernetes error")

		_, err := FindPools(nlm)
		if err == nil {
			t.Fatalf("kubernetes issue did not return an error")
		}
	})
}

func newNodeInPool(t *testing.T, poolName string) corev1.Node {
	t.Helper()

	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"pool": poolName,
			},
		},
	}
}
