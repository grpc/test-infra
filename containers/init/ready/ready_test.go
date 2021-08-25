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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/kubehelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WaitForReadyPods", func() {
	var fastDuration time.Duration
	var slowDuration time.Duration

	var driverPod corev1.Pod
	var serverPod corev1.Pod
	var clientPod corev1.Pod

	BeforeEach(func() {
		fastDuration = 1 * time.Millisecond * timeMultiplier
		slowDuration = 100 * time.Millisecond * timeMultiplier

		driverPod = newTestPod("driver")
		driverRunContainer := kubehelpers.ContainerForName(config.RunContainerName, driverPod.Spec.Containers)
		driverRunContainer.Ports = nil

		serverPod = newTestPod("server")
		serverPod.Status.PodIP = "127.0.0.2"

		clientPod = newTestPod("client")
		clientPod.Status.PodIP = "127.0.0.3"
	})

	It("returns successfully without args", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		podListerMock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{driverPod},
			},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(0, 0),
		}
		podAddresses, nodesInfo, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).ToNot(HaveOccurred())
		Expect(podAddresses).To(BeEmpty())
		Expect(*nodesInfo).To(Equal(NodesInfo{
			Driver: NodeInfo{
				Name:     driverPod.Name,
				PodIP:    driverPod.Status.PodIP,
				NodeName: driverPod.Spec.NodeName,
			},
		}))
	})

	It("timeout reached when no matching pods are found", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()
		clientPod.ObjectMeta.OwnerReferences[0].UID = types.UID("other-test-uid")
		podListerMock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					clientPod,
					driverPod,
				},
			},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(1, 0),
		}
		_, _, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).To(HaveOccurred())
	})

	It("timeout reached when only some matching pods are found", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		podListerMock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					driverPod,
					clientPod,
					// note: missing 2nd client
				},
			},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(2, 0),
		}

		_, _, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).To(HaveOccurred())
	})

	It("timeout reached when no ready pod matches", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		serverPod.Status.ContainerStatuses[0].Ready = false
		podListerMock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					serverPod,
					driverPod,
				},
			},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(0, 1),
		}
		_, _, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).To(HaveOccurred())
	})

	It("does not match the same pod twice", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		podListerMock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					clientPod,
					driverPod,
				},
			},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(2, 0),
		}
		_, _, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).To(HaveOccurred())
	})

	It("returns successfully when all matching pods are found", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		client2Pod := newTestPod("client")
		client2Pod.Name = "client-2"
		client2Pod.Status.PodIP = "127.0.0.4"

		podListerMock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					driverPod,
					clientPod,
					client2Pod,
					serverPod,
				},
			},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(2, 1),
		}

		podAddresses, _, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).ToNot(HaveOccurred())
		Expect(podAddresses).To(Equal([]string{
			fmt.Sprintf("%s:%d", serverPod.Status.PodIP, DefaultDriverPort),
			fmt.Sprintf("%s:%d", clientPod.Status.PodIP, DefaultDriverPort),
			fmt.Sprintf("%s:%d", client2Pod.Status.PodIP, DefaultDriverPort),
		}))
	})

	It("returns with correct ports for matching pods", func() {
		ctx, cancel := context.WithTimeout(context.Background(), slowDuration)
		defer cancel()

		var customPort int32 = 9542
		client2Pod := newTestPod("client")
		client2Pod.Name = "client-2"
		client2PodContainer := kubehelpers.ContainerForName(config.RunContainerName, client2Pod.Spec.Containers)
		client2PodContainer.Ports[0].ContainerPort = customPort

		podListerMock := &PodListerMock{
			PodList: &corev1.PodList{
				Items: []corev1.Pod{
					clientPod,
					client2Pod,
					driverPod,
				},
			},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(2, 0),
		}

		podAddresses, _, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).ToNot(HaveOccurred())
		Expect(podAddresses).To(Equal([]string{
			fmt.Sprintf("%s:%d", clientPod.Status.PodIP, DefaultDriverPort),
			fmt.Sprintf("%s:%d", client2Pod.Status.PodIP, customPort),
		}))
	})

	It("returns error if timeout exceeded", func() {
		ctx, cancel := context.WithTimeout(context.Background(), fastDuration)
		defer cancel()

		podListerMock := &PodListerMock{
			SleepDuration: slowDuration,
			PodList:       &corev1.PodList{},
		}

		loadTestGetterMock := &LoadTestGetterMock{
			Loadtest: newLoadTestWithMultipleClientsAndServers(2, 0),
		}
		_, _, err := WaitForReadyPods(ctx, loadTestGetterMock, podListerMock, "test name")
		Expect(err).To(HaveOccurred())
	})
})

type PodListerMock struct {
	PodList       *corev1.PodList
	SleepDuration time.Duration
	Error         error
	invocation    int
}

var _ PodLister = &PodListerMock{}

func (plm *PodListerMock) List(_ context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	time.Sleep(plm.SleepDuration)

	if plm.Error != nil {
		return nil, plm.Error
	}

	return plm.PodList, nil
}

type LoadTestGetterMock struct {
	Loadtest      *grpcv1.LoadTest
	SleepDuration time.Duration
	Error         error
	invocation    int
}

var _ LoadTestGetter = &LoadTestGetterMock{}

func (lgm *LoadTestGetterMock) Get(_ context.Context, testName string, opts metav1.GetOptions) (*grpcv1.LoadTest, error) {
	time.Sleep(lgm.SleepDuration)

	if lgm.Error != nil {
		return nil, lgm.Error
	}

	return lgm.Loadtest, nil
}
