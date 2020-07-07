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
