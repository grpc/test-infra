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
	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CheckMissingPods", func() {

	var test *grpcv1.LoadTest
	var allRunningPods []*corev1.Pod
	var actualReturn *LoadTestMissing
	var expectedReturn *LoadTestMissing

	BeforeEach(func() {
		test = newLoadTestWithMultipleClientsAndServers()
		allRunningPods = []*corev1.Pod{}
		expectedReturn = &LoadTestMissing{Clients: []grpcv1.Client{}, Servers: []grpcv1.Server{}}
	})

	Context("no pods from the current load test is running", func() {
		BeforeEach(func() {
			for i := 0; i < len(test.Spec.Clients); i++ {
				expectedReturn.Clients = append(expectedReturn.Clients, test.Spec.Clients[i])
			}
			for i := 0; i < len(test.Spec.Servers); i++ {
				expectedReturn.Servers = append(expectedReturn.Servers, test.Spec.Servers[i])
			}
			expectedReturn.Driver = test.Spec.Driver
		})

		It("returns the full pod list from the current load test", func() {
			actualReturn = CheckMissingPods(test, allRunningPods)
			Expect(actualReturn.Clients).To(ConsistOf(expectedReturn.Clients))
			Expect(actualReturn.Servers).To(ConsistOf(expectedReturn.Servers))
			Expect(actualReturn.Driver).To(Equal(expectedReturn.Driver))
		})

		It("sets the number of nodes missing from each pool", func() {
			actualReturn = CheckMissingPods(test, allRunningPods)
			Expect(actualReturn.NodeCountByPool).To(Equal(
				map[string]int{
					"drivers":         1,
					"workers":         6,
					DefaultClientPool: 0,
					DefaultDriverPool: 0,
					DefaultServerPool: 0,
				},
			))
		})
	})

	Context("some of pods from the current load test is running", func() {
		BeforeEach(func() {
			allRunningPods = append(allRunningPods,
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "random-name",
						Labels: map[string]string{
							config.RoleLabel:          "server",
							config.ComponentNameLabel: "server-1",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "random-name",
						Labels: map[string]string{
							config.RoleLabel:          "client",
							config.ComponentNameLabel: "client-2",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "random-name",
						Labels: map[string]string{
							config.RoleLabel:          "driver",
							config.ComponentNameLabel: "driver-1",
						},
					},
				},
			)
			for i := 0; i < len(test.Spec.Clients); i++ {
				if *test.Spec.Clients[i].Name != "client-2" {
					expectedReturn.Clients = append(expectedReturn.Clients, test.Spec.Clients[i])
				}
			}

			for i := 0; i < len(test.Spec.Servers); i++ {
				if *test.Spec.Servers[i].Name != "server-1" {
					expectedReturn.Servers = append(expectedReturn.Servers, test.Spec.Servers[i])
				}
			}
		})

		It("returns the list of pods missing from collection of running pods", func() {
			actualReturn = CheckMissingPods(test, allRunningPods)
			Expect(actualReturn.Clients).To(ConsistOf(expectedReturn.Clients))
			Expect(actualReturn.Servers).To(ConsistOf(expectedReturn.Servers))
			Expect(actualReturn.Driver).To(Equal(expectedReturn.Driver))
		})

		It("sets the number of nodes missing from each pool", func() {
			actualReturn = CheckMissingPods(test, allRunningPods)
			Expect(actualReturn.NodeCountByPool).To(Equal(
				map[string]int{
					"workers":         4,
					DefaultClientPool: 0,
					DefaultDriverPool: 0,
					DefaultServerPool: 0,
				},
			))
		})
	})

	Context("all of pods from the current load test is running", func() {
		BeforeEach(func() {
			allRunningPods = populatePodListWithCurrentLoadTestPod(test)
		})

		It("returns a empty list", func() {
			actualReturn = CheckMissingPods(test, allRunningPods)
			Expect(actualReturn.Clients).To(ConsistOf(expectedReturn.Clients))
			Expect(actualReturn.Servers).To(ConsistOf(expectedReturn.Servers))
			Expect(actualReturn.Driver).To(Equal(expectedReturn.Driver))
		})

		It("sets the number of nodes missing from each pool", func() {
			actualReturn = CheckMissingPods(test, allRunningPods)
			Expect(actualReturn.NodeCountByPool).To(Equal(
				map[string]int{
					DefaultClientPool: 0,
					DefaultDriverPool: 0,
					DefaultServerPool: 0,
				},
			))
		})
	})
})
