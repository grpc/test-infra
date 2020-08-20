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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WaitForReadyPods", func() {
	var fastDuration time.Duration
	var slowDuration time.Duration

	var irrelevantPod corev1.Pod
	var driverPod corev1.Pod
	var serverPod corev1.Pod
	var clientPod corev1.Pod

	BeforeEach(func() {
		fastDuration = 1 * time.Millisecond * timeMultiplier
		slowDuration = 1000 * time.Millisecond * timeMultiplier

		irrelevantPod = corev1.Pod{}

		driverPod = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"role": "driver",
				},
			},
			Status: corev1.PodStatus{
				PodIP: "127.0.0.1",
			},
		}

		serverPod = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"role": "server",
				},
			},
			Status: corev1.PodStatus{
				PodIP: "127.0.0.2",
			},
		}

		clientPod = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"role": "client",
				},
			},
			Status: corev1.PodStatus{
				PodIP: "127.0.0.3",
			},
		}

	})

	It("returns successfully without args", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mock := &PodListerMock{
			PodList: &corev1.PodList{},
		}

		podIPs, err := WaitForReadyPods(ctx, mock, []string{})
		Expect(err).ToNot(HaveOccurred())
		Expect(podIPs).To(BeEmpty())
	})

	It("timeout reached when no matching pods are found", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		mock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{irrelevantPod},
			},
		}

		_, err := WaitForReadyPods(ctx, mock, []string{"hello-anyone-out-there"})
		Expect(err).To(HaveOccurred())
	})

	It("timeout reached when only some matching pods are found", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		mock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					driverPod,
					clientPod,
					// note: missing 2nd client
				},
			},
		}

		_, err := WaitForReadyPods(ctx, mock, []string{"role=driver", "role=client", "role=client"})
		Expect(err).To(HaveOccurred())
	})

	It("timeout reached when pod does not match all labels", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		mock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{driverPod},
			},
		}

		_, err := WaitForReadyPods(ctx, mock, []string{"role=driver,loadtest=loadtest-1"})
		Expect(err).To(HaveOccurred())
	})

	It("returns successfully when all matching pods are found", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		mock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					driverPod,
					serverPod,
					clientPod,
					clientPod,
				},
			},
		}

		podIPs, err := WaitForReadyPods(ctx, mock, []string{
			"role=driver",
			"role=server",
			"role=client",
			"role=client",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(podIPs).To(Equal([]string{
			driverPod.Status.PodIP,
			serverPod.Status.PodIP,
			clientPod.Status.PodIP,
			clientPod.Status.PodIP,
		}))
	})

	It("returns error if timeout exceeded", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fastDuration)
		defer cancel()

		mock := &PodListerMock{
			SleepDuration: slowDuration,
			PodList:       &corev1.PodList{},
		}

		_, err := WaitForReadyPods(ctx, mock, []string{"example"})
		Expect(err).To(HaveOccurred())
	})
})

type PodListerMock struct {
	PodList       *corev1.PodList
	SleepDuration time.Duration
	Error         error
	invocation    int
}

func (plm *PodListerMock) List(opts metav1.ListOptions) (*corev1.PodList, error) {
	time.Sleep(plm.SleepDuration)

	if plm.Error != nil {
		return nil, plm.Error
	}

	return plm.PodList, nil
}
