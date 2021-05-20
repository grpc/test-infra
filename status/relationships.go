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
	corev1 "k8s.io/api/core/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// PodsForLoadTest returns a slice of pointers to pods which belong to a
// specific load test. It accepts the load test to match and a list of all pods
// to consider. If none of the pods match, an empty slice is returned.
func PodsForLoadTest(loadtest *grpcv1.LoadTest, allPods []corev1.Pod) []*corev1.Pod {
	if loadtest == nil {
		return nil
	}

	uid := loadtest.GetUID()

	var pods []*corev1.Pod

	for i := range allPods {
		pod := &allPods[i]
		for _, owner := range pod.GetOwnerReferences() {
			if owner.UID == uid {
				pods = append(pods, pod)
				break
			}
		}
	}

	return pods
}
