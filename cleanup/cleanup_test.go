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

var _ = Describe("Test Environment", func() {
	It("supports creation of load tests", func() {
		err := k8sClient.Create(context.Background(), newLoadTest())
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("quitWorkers", func() {
	var log logr.Logger
	var podList []*corev1.Pod
	var driver *corev1.Pod
	var mockQuit *mockQuitClient

	var clientPending *corev1.Pod
	var serverPending *corev1.Pod
	var clientRunning *corev1.Pod
	var serverRunning *corev1.Pod
	var clientFailed *corev1.Pod
	var serverFailed *corev1.Pod
	var clientUnknown *corev1.Pod
	var serverUnknown *corev1.Pod
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
				Phase: corev1.PodPending,
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
				Phase: corev1.PodPending,
			},
		}

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
				Phase: corev1.PodSucceeded,
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
				Phase: corev1.PodSucceeded,
			},
		}

		clientRunning = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "client-running",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ClientRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}

		serverRunning = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "server-running",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ServerRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}

		clientFailed = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "client-failed",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ClientRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}

		serverFailed = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "server-failed",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ServerRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}

		clientUnknown = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "client-unknown",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ClientRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodUnknown,
			},
		}

		serverUnknown = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "server-unknown",
				Labels: map[string]string{
					config.LoadTestLabel:      "LoadTest",
					config.RoleLabel:          config.ServerRole,
					config.ComponentNameLabel: "name",
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodUnknown,
			},
		}

	})

	It("doesn't send quit to driver", func() {
		podList = append(podList, driver)

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(BeEmpty())
	})

	It("doesn't send quit to Succeeded workers", func() {
		podList = append(podList, clientSucceeded, serverSucceeded)

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(BeEmpty())
	})

	It("doesn't send quit to failed worker", func() {
		podList = append(podList, clientFailed, serverFailed)

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(BeEmpty())
	})

	It("send quit to workers in unknown state", func() {
		podList = append(podList, clientUnknown, serverUnknown)
		expected := []string{"client-unknown", "server-unknown"}

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(Equal(expected))
	})

	It("send quit to workers in running state", func() {
		podList = append(podList, clientRunning, serverRunning)
		expected := []string{"client-running", "server-running"}
		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(Equal(expected))
	})

	It("sends quit to workers in pending state", func() {
		podList = append(podList, clientPending, serverPending)

		expected := []string{"client-pending", "server-pending"}

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(Equal(expected))
	})

	It("sends quit to workers need to be quit", func() {
		podList = append(podList, clientPending, serverPending, clientRunning, serverRunning, clientUnknown, serverUnknown, clientFailed, serverFailed, clientSucceeded, serverSucceeded, driver)

		expected := []string{"client-pending", "server-pending", "client-running", "server-running", "client-unknown", "server-unknown"}

		quitWorkers(ctx, mockQuit, podList, log)
		Expect(mockQuit.called).To(Equal(expected))
	})
})
