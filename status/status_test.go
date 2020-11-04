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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/optional"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StateForContainerStatus", func() {
	var status *corev1.ContainerStatus

	Context("container running", func() {
		BeforeEach(func() {
			status = &corev1.ContainerStatus{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}
		})

		It("returns a pending state and nil exit code", func() {
			state, exitCode := StateForContainerStatus(status)
			Expect(state).To(Equal(Pending))
			Expect(exitCode).To(BeNil())
		})
	})

	Context("container waiting", func() {
		BeforeEach(func() {
			status = &corev1.ContainerStatus{
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{},
				},
			}
		})

		Context("crash detected", func() {
			It("returns an errored state and nil exit code", func() {
				status.State.Waiting.Reason = "CrashLoopBackOff"
				state, exitCode := StateForContainerStatus(status)
				Expect(state).To(Equal(Errored))
				Expect(exitCode).To(BeNil())
			})
		})

		Context("no crash detected", func() {
			It("returns a pending state and nil exit code", func() {
				state, exitCode := StateForContainerStatus(status)
				Expect(state).To(Equal(Pending))
				Expect(exitCode).To(BeNil())
			})
		})
	})

	Context("container terminated", func() {
		BeforeEach(func() {
			status = &corev1.ContainerStatus{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 1,
					},
				},
			}
		})

		Context("successful", func() {
			It("returns a succeeded state and exit code", func() {
				status.State.Terminated.ExitCode = 0

				state, exitCode := StateForContainerStatus(status)
				Expect(state).To(Equal(Succeeded))
				Expect(exitCode).ToNot(BeNil())
				Expect(*exitCode).To(BeEquivalentTo(0))
			})
		})

		Context("unsuccessful", func() {
			It("returns an errored state and exit code", func() {
				status.State.Terminated.ExitCode = 127

				state, exitCode := StateForContainerStatus(status)
				Expect(state).To(Equal(Errored))
				Expect(exitCode).ToNot(BeNil())
				Expect(*exitCode).To(BeEquivalentTo(127))
			})
		})
	})
})

var _ = Describe("StateForPodStatus", func() {
	var podStatus *corev1.PodStatus
	var initContainer1, initContainer2, container *corev1.ContainerStatus

	BeforeEach(func() {
		podStatus = &corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{},
				},
				{
					State: corev1.ContainerState{},
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{},
				},
			},
		}

		initContainer1 = &podStatus.InitContainerStatuses[0]
		initContainer2 = &podStatus.InitContainerStatuses[1]
		container = &podStatus.ContainerStatuses[0]
	})

	Context("init containers running", func() {
		BeforeEach(func() {
			container.State.Waiting = &corev1.ContainerStateWaiting{}
		})

		It("marks pod as pending when init containers are pending", func() {
			// Set the first init container as succeeded to ensure we do not just rely
			// on the first init container's success.
			initContainer1.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}

			initContainer2.State.Running = &corev1.ContainerStateRunning{}

			state, _, _ := StateForPodStatus(podStatus)
			Expect(state).To(Equal(Pending))
		})

		It("marks pod as pending when init containers succeeded", func() {
			initContainer1.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}
			initContainer2.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}

			state, _, _ := StateForPodStatus(podStatus)
			Expect(state).To(Equal(Pending))
		})

		It("marks pod as errored when init containers errored", func() {
			// Set the first init container as succeeded to ensure we do not just rely
			// on the first init container's success.
			initContainer1.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}

			initContainer2.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 127}

			state, reason, _ := StateForPodStatus(podStatus)
			Expect(state).To(Equal(Errored))
			Expect(reason).To(Equal(grpcv1.InitContainerError))
		})
	})

	Context("init containers succeeded", func() {
		It("marks pod as pending when containers are pending", func() {
			initContainer1.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}
			initContainer2.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}

			container.State.Running = &corev1.ContainerStateRunning{}

			state, _, _ := StateForPodStatus(podStatus)
			Expect(state).To(Equal(Pending))
		})

		It("marks pod as succeeded when containers succeeded", func() {
			container.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}

			state, _, _ := StateForPodStatus(podStatus)
			Expect(state).To(Equal(Succeeded))
		})

		It("marks pod as errored when containers errored", func() {
			container.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 127}

			state, reason, _ := StateForPodStatus(podStatus)
			Expect(state).To(Equal(Errored))
			Expect(reason).To(Equal(grpcv1.ContainerError))
		})

		It("marks a pod as pending if not all containers have finished", func() {
			container.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0}
			podStatus.ContainerStatuses = append(podStatus.ContainerStatuses, corev1.ContainerStatus{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			})
			podStatus.ContainerStatuses = append(podStatus.ContainerStatuses, corev1.ContainerStatus{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
				},
			})

			state, _, _ := StateForPodStatus(podStatus)
			Expect(state).To(Equal(Pending))
		})
	})
})

var _ = Describe("ForLoadTest", func() {
	var test *grpcv1.LoadTest
	var pods []*corev1.Pod
	var driverPod, serverPod, clientPod *corev1.Pod

	BeforeEach(func() {
		test = &grpcv1.LoadTest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-for-unit-tests",
			},
			Spec: grpcv1.LoadTestSpec{
				Driver: &grpcv1.Driver{
					Component: grpcv1.Component{
						Name: optional.StringPtr("driver"),
						Run: grpcv1.Run{
							Image: optional.StringPtr("fake-driver-image"),
						},
					},
				},
				Servers: []grpcv1.Server{
					{
						Component: grpcv1.Component{
							Name: optional.StringPtr("server-1"),
							Run: grpcv1.Run{
								Image: optional.StringPtr("fake-server-image"),
							},
						},
					},
				},
				Clients: []grpcv1.Client{
					{
						Component: grpcv1.Component{
							Name: optional.StringPtr("client-1"),
							Run: grpcv1.Run{
								Image: optional.StringPtr("fake-client-image"),
							},
						},
					},
				},
			},
		}

		pods = []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "driver",
					Labels: map[string]string{
						config.LoadTestLabel:      test.Name,
						config.RoleLabel:          config.DriverRole,
						config.ComponentNameLabel: "driver",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "server-1",
					Labels: map[string]string{
						config.LoadTestLabel:      test.Name,
						config.RoleLabel:          config.ServerRole,
						config.ComponentNameLabel: "server-1",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "client-1",
					Labels: map[string]string{
						config.LoadTestLabel:      test.Name,
						config.RoleLabel:          config.ClientRole,
						config.ComponentNameLabel: "client-1",
					},
				},
			},
		}

		driverPod = pods[0]
		serverPod = pods[1]
		clientPod = pods[2]

		_ = driverPod
		_ = serverPod
		_ = clientPod
	})

	It("sets start time when unset", func() {
		testStart := metav1.Now()

		status := ForLoadTest(test, pods)

		Expect(status.StartTime).ToNot(BeNil())
		Expect(testStart.Before(status.StartTime)).To(BeTrue())
	})

	It("does not override start time when set", func() {
		fakeStartTime := metav1.Now()
		test.Status.StartTime = &fakeStartTime

		status := ForLoadTest(test, pods)

		Expect(status.StartTime).To(Equal(&fakeStartTime))
	})

	It("sets succeeded state when driver pod succeeded", func() {
		driverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
				},
			},
		}

		serverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		clientPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		status := ForLoadTest(test, pods)

		Expect(status.State).To(BeEquivalentTo(grpcv1.Succeeded))
	})

	It("does not set succeeded state when worker pods succeeded", func() {
		driverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		serverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
				},
			},
		}

		clientPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 0},
				},
			},
		}

		status := ForLoadTest(test, pods)

		Expect(status.State).ToNot(BeEquivalentTo(grpcv1.Succeeded))
	})

	It("sets failed state when driver pod errored", func() {
		driverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 1},
				},
			},
		}

		serverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		clientPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		status := ForLoadTest(test, pods)

		Expect(status.State).To(BeEquivalentTo(grpcv1.Failed))
	})

	It("sets errored state when driver pod init container errored", func() {
		driverPod.Status.InitContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 127},
				},
			},
		}

		driverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{},
				},
			},
		}

		serverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		clientPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		status := ForLoadTest(test, pods)

		Expect(status.State).To(BeEquivalentTo(grpcv1.Errored))
	})

	It("sets errored state when worker pod errored", func() {
		driverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		serverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		clientPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 1},
				},
			},
		}

		status := ForLoadTest(test, pods)

		Expect(status.State).To(BeEquivalentTo(grpcv1.Errored))
	})

	It("sets stop time when unset", func() {
		testStart := metav1.Now()

		driverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		serverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		clientPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 1},
				},
			},
		}

		status := ForLoadTest(test, pods)

		Expect(status.StopTime).ToNot(BeNil())
		Expect(testStart.Before(status.StopTime)).To(BeTrue())
	})

	It("does not override stop time when set", func() {
		driverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		serverPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			},
		}

		clientPod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{ExitCode: 1},
				},
			},
		}

		stopTime := optional.CurrentTimePtr()
		test.Status.StopTime = stopTime

		status := ForLoadTest(test, pods)

		Expect(status.StopTime).ToNot(BeNil())
		Expect(*status.StopTime).To(Equal(*stopTime))
	})

	It("sets initializing state when pods are missing", func() {
		pods = pods[1:] // remove the driver from the world

		status := ForLoadTest(test, pods)

		Expect(status.State).To(BeEquivalentTo(grpcv1.Initializing))
	})
})
