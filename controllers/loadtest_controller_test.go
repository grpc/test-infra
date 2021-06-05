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
	"reflect"

	"github.com/google/uuid"
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
	kind := reflect.TypeOf(grpcv1.LoadTest{}).Name()
	gvk := grpcv1.GroupVersion.WithKind(kind)
	controllerRef := metav1.NewControllerRef(test.DeepCopy().GetObjectMeta(), gvk)
	pod.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
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

// deleteTestPods is a helper that attempts to clean-up all pods for load test.
// It ignores any errors, since not all pods may exist that it attempts to
// delete.
func deleteTestPods(test *grpcv1.LoadTest) {
	builder := podbuilder.New(defaults, test)
	for _, server := range test.Spec.Servers {
		pod, err := builder.PodForServer(&server)
		if err != nil {
			k8sClient.Delete(context.Background(), pod)
		}
	}
	for _, client := range test.Spec.Clients {
		pod, err := builder.PodForClient(&client)
		if err != nil {
			k8sClient.Delete(context.Background(), pod)
		}
	}
	pod, err := builder.PodForDriver(test.Spec.Driver)
	if err != nil {
		k8sClient.Delete(context.Background(), pod)
	}
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

	It("does not change the test status after termination", func() {
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

	It("does not create nodes if there are inadequate machines", func() {
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "drivers",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "workers-a",
					capacity: 1, // only 1 node!
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		Consistently(func() (int, error) {
			foundPodCount := 0

			list := new(corev1.PodList)
			if err := k8sClient.List(context.Background(), list, client.InNamespace(test.Namespace)); err != nil {
				return 0, err
			}

			for i := range list.Items {
				item := &list.Items[i]
				for _, owner := range item.GetOwnerReferences() {
					if owner.UID == test.GetUID() {
						foundPodCount++
						break
					}
				}
			}

			return foundPodCount, nil
		}).Should(Equal(0))

		// no pods should be created, but clean-up just in case
		deleteTestPods(test)
	})

	It("does not schedule pods for tests that will fight for machines", func() {
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "drivers",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "workers-a",
					capacity: 3,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name

		test2 := test.DeepCopy()
		test2.Name = "test-2"

		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())
		Expect(k8sClient.Create(context.Background(), test2)).To(Succeed())

		Eventually(func() bool {
			list := new(corev1.PodList)
			if err := k8sClient.List(context.Background(), list, client.InNamespace(test.Namespace)); err != nil {
				return false
			}

			return len(list.Items) > 0
		}).Should(BeTrue())

		Consistently(func() (int, error) {
			runningTestUIDSet := make(map[types.UID]bool)

			list := new(corev1.PodList)
			if err := k8sClient.List(context.Background(), list, client.InNamespace(test.Namespace)); err != nil {
				return 0, err
			}

			for i := range list.Items {
				item := &list.Items[i]
				for _, owner := range item.GetOwnerReferences() {
					if _, ok := runningTestUIDSet[owner.UID]; !ok {
						runningTestUIDSet[owner.UID] = true
					}
				}
			}

			// return the number of running tests, which should be 1
			return len(runningTestUIDSet), nil
		}).Should(Equal(1))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
		deleteTestPods(test2)
	})

	It("does not block a node from scheduling due to a completed pod", func() {
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "completed-test-drivers",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "completed-test-workers",
					capacity: 2,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name

		test2 := test.DeepCopy()
		test2.Name = uuid.New().String()

		builder := podbuilder.New(defaults, test)
		for _, server := range test.Spec.Servers {
			pod, err := builder.PodForServer(&server)
			Expect(err).ToNot(HaveOccurred())
			pod.Labels[config.PoolLabel] = cluster.pools[1].name
			Expect(k8sClient.Create(context.Background(), pod)).To(Succeed())
			pod.Status.Phase = corev1.PodSucceeded
			Expect(k8sClient.Status().Update(context.Background(), pod)).To(Succeed())
		}
		for _, client := range test.Spec.Clients {
			pod, err := builder.PodForClient(&client)
			Expect(err).ToNot(HaveOccurred())
			pod.Labels[config.PoolLabel] = cluster.pools[1].name
			Expect(k8sClient.Create(context.Background(), pod)).To(Succeed())
			pod.Status.Phase = corev1.PodSucceeded
			Expect(k8sClient.Status().Update(context.Background(), pod)).To(Succeed())
		}
		pod, err := builder.PodForDriver(test.Spec.Driver)
		Expect(err).ToNot(HaveOccurred())
		pod.Labels[config.PoolLabel] = cluster.pools[0].name
		Expect(k8sClient.Create(context.Background(), pod)).To(Succeed())
		pod.Status.Phase = corev1.PodFailed
		Expect(k8sClient.Status().Update(context.Background(), pod)).To(Succeed())

		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		test2.Name = "completed-test-2"
		Expect(k8sClient.Create(context.Background(), test2)).To(Succeed())

		Eventually(func() (int, error) {
			runningTestUIDSet := make(map[types.UID]bool)

			list := new(corev1.PodList)
			if err := k8sClient.List(context.Background(), list, client.InNamespace(test.Namespace)); err != nil {
				return 0, err
			}

			for i := range list.Items {
				item := &list.Items[i]
				for _, owner := range item.GetOwnerReferences() {
					if _, ok := runningTestUIDSet[owner.UID]; !ok {
						runningTestUIDSet[owner.UID] = true
					}
				}
			}

			// return the number of running tests, which should be 2 (since one is completed)
			return len(runningTestUIDSet), nil
		}).Should(Equal(2))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
		deleteTestPods(test2)
	})

	It("creates correct number of pods when all are missing", func() {
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "drivers-2",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "workers-2",
					capacity: 7,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name
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
				for _, owner := range item.GetOwnerReferences() {
					if owner.UID == test.GetUID() {
						foundPodCount++
					}
				}
			}

			return foundPodCount, nil
		}).Should(Equal(expectedPodCount))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
	})

	It("updates the test status when client pods terminate with errors", func() {
		By("creating a fake environment with errored pods")
		runningState := corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		}
		errorState := corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
			},
		}

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		var err error
		for i := range testSpec.Servers {
			pod, err = builder.PodForServer(&testSpec.Servers[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod, err = builder.PodForClient(&testSpec.Clients[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, errorState)).To(Succeed())

		}
		if testSpec.Driver != nil {
			pod, err = builder.PodForDriver(testSpec.Driver)
			Expect(err).ToNot(HaveOccurred())
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

		By("ensuring the test state becomes errored")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Errored))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
	})

	It("updates the test status when driver pod terminated with errors", func() {
		By("creating a fake environment with errored pods")

		var err error
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "drivers",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "workers-a",
					capacity: 2,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name

		runningState := corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		}
		errorState := corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
			},
		}

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		for i := range testSpec.Servers {
			pod, err = builder.PodForServer(&testSpec.Servers[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod, err = builder.PodForClient(&testSpec.Clients[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())

		}
		if testSpec.Driver != nil {
			pod, err = builder.PodForDriver(testSpec.Driver)
			Expect(err).ToNot(HaveOccurred())
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

		By("ensuring the test state becomes errored")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Errored))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
	})

	It("updates the test status when server pods terminate with errors", func() {
		By("creating a fake environment with errored pods")

		var err error
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "drivers",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "workers-a",
					capacity: 2,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name
		runningState := corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		}
		errorState := corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
			},
		}

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		for i := range testSpec.Servers {
			pod, err = builder.PodForServer(&testSpec.Servers[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, errorState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod, err = builder.PodForClient(&testSpec.Clients[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())

		}
		if testSpec.Driver != nil {
			pod, err = builder.PodForDriver(testSpec.Driver)
			Expect(err).ToNot(HaveOccurred())
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

		By("ensuring the test state becomes errored")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Errored))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
	})

	It("updates the test status when pods are running", func() {
		By("creating a fake environment with running pods")

		var err error
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "drivers",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "workers-a",
					capacity: 2,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name
		runningState := corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		}

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		for i := range testSpec.Servers {
			pod, err = builder.PodForServer(&testSpec.Servers[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod, err = builder.PodForClient(&testSpec.Clients[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, runningState)).To(Succeed())

		}
		if testSpec.Driver != nil {
			pod, err = builder.PodForDriver(testSpec.Driver)
			Expect(err).ToNot(HaveOccurred())
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

		By("ensuring the test state becomes running")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Running))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
	})

	It("updates the test status when pods terminate successfully", func() {
		By("creating a fake environment with finished pods")

		var err error
		clusterCfg := &testClusterConfig{
			pools: []*testPool{
				{
					name:     "drivers",
					capacity: 1,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Driver: "true",
					},
				},
				{
					name:     "workers-a",
					capacity: 2,
					labels: map[string]string{
						defaults.DefaultPoolLabels.Client: "true",
						defaults.DefaultPoolLabels.Server: "true",
					},
				},
			},
		}
		cluster, err := createCluster(context.Background(), k8sClient, clusterCfg)
		Expect(err).ToNot(HaveOccurred())
		defer deleteCluster(context.Background(), k8sClient, cluster)

		test.Spec.Driver.Pool = &cluster.pools[0].name
		test.Spec.Clients[0].Pool = &cluster.pools[1].name
		test.Spec.Servers[0].Pool = &cluster.pools[1].name
		successState := corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
			},
		}

		By("creating the load test")
		Expect(k8sClient.Create(context.Background(), test)).To(Succeed())

		builder := podbuilder.New(newDefaults(), test)
		testSpec := &test.Spec
		var pod *corev1.Pod
		for i := range testSpec.Servers {
			pod, err = builder.PodForServer(&testSpec.Servers[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, successState)).To(Succeed())
		}
		for i := range testSpec.Clients {
			pod, err = builder.PodForClient(&testSpec.Clients[i])
			Expect(err).ToNot(HaveOccurred())
			Expect(createPod(pod, test)).To(Succeed())
			Expect(updatePodWithContainerState(pod, successState)).To(Succeed())
		}
		if testSpec.Driver != nil {
			pod, err = builder.PodForDriver(testSpec.Driver)
			Expect(err).ToNot(HaveOccurred())
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

		By("ensuring the test state becomes succeeded")
		Eventually(func() (grpcv1.LoadTestState, error) {
			fetchedTest := new(grpcv1.LoadTest)
			if err := k8sClient.Get(context.Background(), namespacedName, fetchedTest); err != nil {
				return grpcv1.Unknown, err
			}
			return fetchedTest.Status.State, nil
		}).Should(Equal(grpcv1.Succeeded))

		// clean-up all pods for hermetic purposes
		deleteTestPods(test)
	})
})
