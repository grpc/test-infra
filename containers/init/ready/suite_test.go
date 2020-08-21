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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// timeMultiplier provides a way to increase or decrease the timeouts for each
// test.
const timeMultiplier = 1

func TestReady(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ready Suite")
}

func newTestPod(role string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: role,
			Labels: map[string]string{
				"role": role,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "run",
					Ports: []corev1.ContainerPort{
						{
							Name:          "driver",
							Protocol:      corev1.ProtocolTCP,
							ContainerPort: DefaultDriverPort,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			PodIP: "127.0.0.1",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
				},
			},
		},
	}
}
