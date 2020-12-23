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

package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/podbuilder"
	"github.com/grpc/test-infra/status"
)

// createPod creates a pod resource, given a pod pointer and a test pointer.
func createPod(pod *corev1.Pod, test *grpcv1.LoadTest) error {
	// TODO: Get the controllerRef to work here.
	// kind := reflect.TypeOf(grpcv1.LoadTest{}).Name()
	// gvk := grpcv1.GroupVersion.WithKind(kind)
	// controllerRef := metav1.NewControllerRef(test, gvk)
	// pod.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
	return k8sClient.Create(context.Background(), pod)
}

// updatePodWithContainerState changes the container state in the status of a
// pod resource that already exists on the cluster. This is useful for testing
// different failing, running and succeeding states.
func updatePodWithContainerState(pod *corev1.Pod, containerState corev1.ContainerState) error {
	status := &pod.Status
	status.ContainerStatuses = []corev1.ContainerStatus{
		{
			State: containerState,
		},
	}
	return k8sClient.Status().Update(context.Background(), pod)
}

var _ = Describe("LoadTest controller", func() {
	var test *grpcv1.LoadTest
	var namespacedName types.NamespacedName

	BeforeEach(func() {
		test = newLoadTest()
		namespacedName = types.NamespacedName{
			Name:      test.Name,
			Namespace: test.Namespace,
		}
	})

	It("does not change the status after termination", func() {
		now := metav1.Now()
		test.Status = grpcv1.LoadTestStatus{
			State:     grpcv1.Succeeded,
			StartTime: &now,
			StopTime:  &now,
		}
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())
		Expect(k8sClient.Status().Update(context.Background(), test)).To(Succeed())

		getTestStatus := func() (grpcv1.LoadTestStatus, error) {
			fetchedTest := new(grpcv1.LoadTest)
			err := k8sClient.Get(context.Background(), namespacedName, fetchedTest)
			if err != nil {
				return grpcv1.LoadTestStatus{}, err
			}
			return fetchedTest.Status, nil
		}

		By("ensuring we can eventually get the created status")
		Eventually(getTestStatus).Should(Equal(test.Status))

		By("checking that the expected status remains unchanged")
		Consistently(getTestStatus).Should(Equal(test.Status))
	})

	It("creates a scenarios ConfigMap", func() {
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		type expectedFields struct {
			name          string
			namespace     string
			scenariosJSON string
			owner         string
		}
		getConfigMapFields := func() (expectedFields, error) {
			cfgMap := new(corev1.ConfigMap)
			err := k8sClient.Get(context.Background(), namespacedName, cfgMap)

			var owner string
			if len(cfgMap.OwnerReferences) > 0 {
				owner = cfgMap.OwnerReferences[0].Name
			}
			return expectedFields{
				name:          cfgMap.Name,
				namespace:     cfgMap.Namespace,
				scenariosJSON: cfgMap.Data["scenarios.json"],
				owner:         owner,
			}, err
		}

		By("checking that the ConfigMap was created correctly")
		Eventually(getConfigMapFields).Should(Equal(expectedFields{
			name:          test.Name,
			namespace:     test.Namespace,
			scenariosJSON: test.Spec.ScenariosJSON,
			owner:         test.Name,
		}))
	})

	It("creates correct number of pods when all are missing", func() {
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		expectedPodCount := 0
		missingPods := status.CheckMissingPods(test, []*corev1.Pod{})
		for range missingPods.Servers {
			expectedPodCount++
		}
		for range missingPods.Clients {
			expectedPodCount++
		}
		if missingPods.Driver != nil {
			expectedPodCount++
		}

		Eventually(func() (int, error) {
			foundPodCount := 0

			list := new(corev1.PodList)
			if err := k8sClient.List(context.Background(), list, client.InNamespace(test.Namespace)); err != nil {
				return 0, err
			}

			for i := range list.Items {
				item := &list.Items[i]
				if item.Labels[config.LoadTestLabel] == test.Name {
					foundPodCount++
				}
			}

			return foundPodCount, nil
		}).Should(Equal(expectedPodCount))
	})

	It("updates the status when pods terminate with errors", func() {
		By("creating a fake environment with errored pods")
		errorState := corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
			},
		}
		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		for i := range testSpec.Servers {
			pod = builder.PodForServer(&testSpec.Servers[i])
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, errorState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod = builder.PodForClient(&testSpec.Clients[i])
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, errorState)).To(Succeed())

		}
		if testSpec.Driver != nil {
			pod = builder.PodForDriver(testSpec.Driver)
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, errorState)).To(Succeed())
		}

		By("waiting for one of the pods to eventually be fetchable")
		Eventually(func() (*corev1.Pod, error) {
			podNamespacedName := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
			fetchedPod := new(corev1.Pod)
			if err := k8sClient.Get(context.Background(), podNamespacedName, fetchedPod); err != nil {
				return nil, err
			}
			return fetchedPod, nil
		}).ShouldNot(BeNil())

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		By("ensuring the test state becomes errored")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Errored))
	})

	It("updates the status when pods are running", func() {
		By("creating a fake environment with running pods")
		runningState := corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		}
		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		for i := range testSpec.Servers {
			pod = builder.PodForServer(&testSpec.Servers[i])
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod = builder.PodForClient(&testSpec.Clients[i])
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())

		}
		if testSpec.Driver != nil {
			pod = builder.PodForDriver(testSpec.Driver)
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())
		}

		By("waiting for one of the pods to eventually be fetchable")
		Eventually(func() (*corev1.Pod, error) {
			podNamespacedName := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
			fetchedPod := new(corev1.Pod)
			if err := k8sClient.Get(context.Background(), podNamespacedName, fetchedPod); err != nil {
				return nil, err
			}
			return fetchedPod, nil
		}).ShouldNot(BeNil())

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		By("ensuring the test state becomes running")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Running))
	})

	It("updates the status when pods terminate successfully", func() {
		By("creating a fake environment with finished pods")
		successState := corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
			},
		}
		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		for i := range testSpec.Servers {
			pod = builder.PodForServer(&testSpec.Servers[i])
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, successState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod = builder.PodForClient(&testSpec.Clients[i])
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, successState)).To(Succeed())

		}
		if testSpec.Driver != nil {
			pod = builder.PodForDriver(testSpec.Driver)
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, successState)).To(Succeed())
		}

		By("waiting for one of the pods to eventually be fetchable")
		Eventually(func() (*corev1.Pod, error) {
			podNamespacedName := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
			fetchedPod := new(corev1.Pod)
			if err := k8sClient.Get(context.Background(), podNamespacedName, fetchedPod); err != nil {
				return nil, err
			}
			return fetchedPod, nil
		}).ShouldNot(BeNil())

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		By("ensuring the test state becomes succeeded")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Succeeded))
	})
})
