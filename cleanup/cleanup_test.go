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

package cleanup

import (
	"context"

	"github.com/go-logr/logr"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/grpc/test-infra/config"
)

var _ = Describe("quitWorkers", func() {
	var log logr.Logger
	var podList []*corev1.Pod
	var driver *corev1.Pod
	var mockQuit *mockQuitClient

	var clientPending *corev1.Pod
	var serverPending *corev1.Pod
	var clientErrored *corev1.Pod
	var serverErrored *corev1.Pod
	var clientSucceeded *corev1.Pod
	var serverSucceeded *corev1.Pod
	var ctx context.Context

	BeforeEach(func() {
		log = ctrl.Log.WithName("CleanupAgent").WithName("LoadTest")
		mockQuit = &mockQuitClient{}
		ctx = context.Background()

		podList = []*corev1.Pod{}

		driver = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "driver",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.DriverRole,
					config.ComponentNameLabel: "name",
				},
			},
		}

		// This is one of the possibilities where the pod would be marked as
		// pending. Testing other possibilities is out of scope of this test, and
		// done elsewhere.
		clientPending = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "client-pending",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ClientRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{},
					},
				},
			},
		}

		serverPending = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "server-pending",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ServerRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{},
					},
				},
			},
		}

		// This is one of the possibilities where the pod would be marked as
		// succeeded. Testing other possibilities is out of the scope of this
		// test, and is done elsewhere.
		clientSucceeded = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "client-succeeded",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ClientRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
			},
		}

		serverSucceeded = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "server-succeeded",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ServerRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
			},
		}

		// This is one of the possibilities where the pod would be marked as
		// errored. Testing other possibilities is out of scope of this test, and
		// done elsewhere.
		clientErrored = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "client-errored",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ClientRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 127},
						},
					},
				},
			},
		}

		serverErrored = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "server-errored",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ServerRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
						},
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: 127},
						},
					},
				},
			},
		}

	})

	It("doesn't send callQuitter to driver", func() {
		podList = append(podList, driver)

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(BeEmpty())
	})

	It("doesn't send callQuitter to succeeded workers", func() {
		podList = append(podList, clientSucceeded, serverSucceeded)

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(BeEmpty())
	})

	It("doesn't send callQuitter to errored worker", func() {
		podList = append(podList, clientErrored, serverErrored)

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(BeEmpty())
	})

	It("sends callQuitter to workers in pending state", func() {
		podList = append(podList, clientPending, serverPending)

		expected := []string{"client-pending", "server-pending"}

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(Equal(expected))
	})

	It("sends callQuitter only to workers that need to be callQuitter", func() {
		podList = append(podList, clientPending, serverPending, clientErrored, serverErrored, clientSucceeded, serverSucceeded, driver)

		expected := []string{"client-pending", "server-pending"}

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(Equal(expected))
	})
})
