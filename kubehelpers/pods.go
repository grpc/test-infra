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

package kubehelpers

import (
	corev1 "k8s.io/api/core/v1"
)

// ContainerForName accepts a string and a slice of containers. It returns a
// pointer to the container with a name that matches the string. If no names
// match, it returns nil.
func ContainerForName(name string, containers []corev1.Container) *corev1.Container {
	for i := range containers {
		container := &containers[i]

		if container.Name == name {
			return container
		}
	}

	return nil
}
