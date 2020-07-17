// Copyright 2020 gRPC authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"fmt"
	"os"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc/test-infra/benchmarks/svc/types"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

const driverPort int32 = 10000
const serverPort int32 = 10010

// gcpSecret is the name of a Kubernetes secret which contains a security
// key for a GCP service account. This key is used to save results. The
// value of this variable will be set to the value of the $GCP_KEY_SECRET
// environment variable at runtime. If empty, there are no adverse effects.
var gcpSecret string

func init() {
	gcpSecret = os.Getenv("GCP_KEY_SECRET")
}

func makePod(loadtest *grpcv1.LoadTest, component *grpcv1.Component, role string, index int) *apiv1.Pod {
	name := fmt.Sprintf("%s-%s-%d", loadtest.Name, role, index)
	mainContainerName := fmt.Sprintf("%s-main", name)

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: loadtest.Namespace,
			Labels: map[string]string{
				LoadTestLabel: loadtest.Name,
				RoleLabel:     role,
				"repulsive":   "true",
			},
		},
		Spec: apiv1.PodSpec{
			NodeSelector: map[string]string{
				// TODO: Add webhook to preset pools
				"pool": *component.Pool,
			},
			Containers: []apiv1.Container{
				{
					Name:  mainContainerName,
					Image: *component.Run.Image,
					Ports: []apiv1.ContainerPort{
						{
							Name:          "driver-port",
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: driverPort,
						},
					},
				},
			},
			Affinity: &apiv1.Affinity{
				PodAntiAffinity: &apiv1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []apiv1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "generated",
										Operator: metav1.LabelSelectorOpExists,
									},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
			RestartPolicy: "Never",
		},
	}

	spec := &pod.Spec
	mainContainer := &spec.Containers[0]

	for _, e := range component.Run.Env {
		mainContainer.Env = append(mainContainer.Env, apiv1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	if role == DriverRole {
		if gcpSecret != "" {
			volumeName := fmt.Sprintf("%s-%s", name, gcpSecret)
			volumeMountPath := "/var/secrets/google"
			spec.Volumes = append(spec.Volumes, apiv1.Volume{
				Name: volumeName,
				VolumeSource: apiv1.VolumeSource{
					Secret: &apiv1.SecretVolumeSource{
						SecretName: gcpSecret,
					},
				},
			})
			mainContainer.VolumeMounts = append(mainContainer.VolumeMounts, apiv1.VolumeMount{
				Name:      volumeName,
				MountPath: volumeMountPath,
			})
			mainContainer.Env = append(mainContainer.Env, apiv1.EnvVar{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: volumeMountPath + "/key.json",
			})
		}

		// mainContainer.Env = append(mainContainer.Env, apiv1.EnvVar{
		// 	Name:  "SCENARIO_JSON",
		// 	Value: scenarioJSON(session),
		// })

	} else {
		mainContainer.Env = append(mainContainer.Env, apiv1.EnvVar{
			Name:  "WORKER_KIND",
			Value: role,
		})
	}

	return pod
}

func scenarioJSON(session *types.Session) string {
	marshaler := &jsonpb.Marshaler{
		Indent:      "",
		EnumsAsInts: true,
		OrigName:    true,
	}

	json, err := marshaler.MarshalToString(session.Scenario)
	if err != nil {
		return ""
	}

	return json
}

func getComponents(loadtest *grpcv1.LoadTest) []*grpcv1.Component {
	spec := &loadtest.Spec

	if spec.Driver.Pool == nil {
		spec.Driver.Pool = &DefaultDriverPool
	}

	components := []*grpcv1.Component{&spec.Driver.Component}

	for _, server := range spec.Servers {
		server.Pool = &DefaultWorkerPool
		components = append(components, &server.Component)
	}

	for _, client := range spec.Clients {
		client.Pool = &DefaultWorkerPool
		components = append(components, &client.Component)
	}

	return components
}

type NodePool struct {
	Nodes []*corev1.Node
	Available int
	Capacity int
}

type LoadTestGraph struct {
	name *string
	test *grpcv1.Test
	driver *corev1.Pod
	clients []*corev1.Pod
	servers []*corev1.Pod
}

type ClusterGraph struct {
	nodes []*corev1.Node
	pools map[string]*NodePool
	pods []*corev1.Pod
	allTests []*grpcv1.LoadTest
	pendingTests map[string]bool
	testGraph *LoadTestGraph
}

func New(nodeList *corev1.nodeList, podList *corev1.PodList, loadTestList *grpcv1.loadTestList, currentTestName string) *ClusterGraph {
	graph := &ClusterGraph{
		pools: make(map[string]*NodePool),
		pendingTests: make(map[string]bool),
	}

	graph.AddNodes(nodeList)
	graph.AddPods(podList)
	graph.AddLoadTests(loadTestList)

	return graph
}

func (c *ClusterGraph) AddNodes(nodes []corev1.Node) {
	for _, node := range nodes {
		c.nodes = append(c.nodes, node)

		labels := node.ObjectMeta.Labels
		if labels == nil {
			continue
		}

		poolName, ok := labels["pool"]
		if !ok {
			continue
		}

		pool, ok := c.pools[poolName]
		if !ok {
			c.pools[poolName] = &NodePool{}
		}

		pool.Available++
		pool.Capacity++
		pool.Nodes = append(pool.Nodes, &node)
	}
}

func (c *ClusterGraph) AddPods(pods []corev1.Pod) {
	for _, pod := range pods {
		if name, ok := pod.Labels[LoadTestLabel]; ok {
			c.pendingTests[name] = true
		}
	}
}

func (c *ClusterGraph) AddLoadTests(loadtests []grpcv1.LoadTest) error {
	for _, loadtest := range loadtests {
		if !c.IsPending(&loadTest) {
			continue
		}

		components := getComponents(&loadtest)
		for _, component := range components {
			if err := c.addAvailableNodes(*component.Pool, -1); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ClusterGraph) Fits(loadtest *grpcv1.LoadTest) bool {
	// build up a map of pool names to number of nodes that the test requires
	requirements := make(map[string]int)

	components := getComponents(&loadTest)
	for _, component := range components {
		c, ok := requirements[*component.Pool]
		if !ok {
			requirements[*component.Pool] = 1
		} else {
			requirements[*component.Pool] = c + 1
		}
	}

	for poolName, requiredNodes := range requirements {
		pool, ok := c.pools[poolName]
		if !ok {
			return false
		}

		if requiredNodes > pool.Available {
			return false
		}
	}

	return true
}

func (c *ClusterGraph) IsPending(loadtest *grpcv1.LoadTest) bool {
	_, ok := c.pendingTests[loadtest.Name]
	return ok
}

func (c *ClusterGraph) addAvailableNodes(poolName string, count int) error {
	pool, ok := c.pools[poolName]
	if !ok {
		return fmt.Errorf("pool %q does not exist", pool)
	}

	nowAvailable := pool.Available + count

	if nowAvailable < 0 {
		return fmt.Errorf("pool %q availability cannot drop below zero", poolName)
	}
	if nowAvailable > pool.Capacity {
		return fmt.Errorf("pool %q availability cannot exceed its capacity (%d nodes)", poolName, pool.Capacity)
	}

	pool.Available = nowAvailable
	return nil
}
