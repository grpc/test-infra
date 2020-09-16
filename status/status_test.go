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
	})
})
