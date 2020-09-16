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
