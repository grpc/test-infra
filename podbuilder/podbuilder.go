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

package podbuilder

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/kubehelpers"
)

// errNoPool is the base error when a PodBuilder cannot determine the pool for
// a pod.
var errNoPool = errors.New("pool is missing")

// addReadyInitContainer configures a ready init container. This container is
// meant to wait for workers to become ready, writing the IP address and port of
// these workers to a file. This file is then shared over a volume with the
// driver's run container.
//
// This method also sets the $QPS_WORKERS_FILE environment variable on the
// driver's run container. Its value will point to the aforementioned, shared
// file.
func addReadyInitContainer(defs *config.Defaults, test *grpcv1.LoadTest, podspec *corev1.PodSpec, container *corev1.Container) {
	if defs == nil || podspec == nil || container == nil {
		return
	}

	readyContainer := newReadyContainer(defs, test)
	podspec.InitContainers = append(podspec.InitContainers, readyContainer)

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "QPS_WORKERS_FILE",
		Value: config.ReadyOutputFile,
	})

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      config.ReadyVolumeName,
		MountPath: config.ReadyMountPath,
	})

	podspec.Volumes = append(podspec.Volumes, corev1.Volume{
		Name: config.ReadyVolumeName,
	})
}

// newReadyContainer constructs a container using the default ready container
// image. If defaults parameter is nil, an empty container is returned.
func newReadyContainer(defs *config.Defaults, test *grpcv1.LoadTest) corev1.Container {
	if defs == nil {
		return corev1.Container{}
	}

	var args []string
	args = append(args, test.GetName())

	return corev1.Container{
		Name:    config.ReadyInitContainerName,
		Image:   defs.ReadyImage,
		Command: []string{"ready"},
		Args:    args,
		Env: []corev1.EnvVar{
			{
				Name:  "READY_OUTPUT_FILE",
				Value: config.ReadyOutputFile,
			},
			{
				Name:  "READY_TIMEOUT",
				Value: fmt.Sprintf("%d%s", test.Spec.TimeoutSeconds, "s"),
			},
			{
				Name:  "METADATA_OUTPUT_FILE",
				Value: config.ReadyMetadataOutputFile,
			},
			{
				Name:  "NODE_INFO_OUTPUT_FILE",
				Value: config.ReadyNodeInfoOutputFile,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      config.ReadyVolumeName,
				MountPath: config.ReadyMountPath,
			},
		},
	}
}

// PodBuilder constructs pods for a test's driver, server and client.
type PodBuilder struct {
	test     *grpcv1.LoadTest
	defaults *config.Defaults
	name     string
	role     string
	pool     string
	clone    *grpcv1.Clone
	build    *grpcv1.Build
	run      []corev1.Container
}

// New creates a PodBuilder instance. It accepts and uses defaults and a test to
// predictably construct pods.
func New(defaults *config.Defaults, test *grpcv1.LoadTest) *PodBuilder {
	return &PodBuilder{
		test:     test,
		defaults: defaults,
	}
}

// PodForClient accepts a pointer to a client and returns a pod for it.
func (pb *PodBuilder) PodForClient(client *grpcv1.Client) (*corev1.Pod, error) {
	pb.name = safeStrUnwrap(client.Name)
	pb.role = config.ClientRole
	pb.pool = safeStrUnwrap(client.Pool)
	pb.clone = client.Clone
	pb.build = client.Build
	pb.run = client.Run

	pod := pb.newPod()

	nodeSelector := make(map[string]string)
	if client.Pool != nil {
		nodeSelector["pool"] = *client.Pool
	} else if pb.defaults.DefaultPoolLabels != nil && pb.defaults.DefaultPoolLabels.Client != "" {
		nodeSelector[pb.defaults.DefaultPoolLabels.Client] = "true"
	} else {
		return nil, errors.Wrapf(errNoPool, "could not determine pool for client %q (no explicit value or default)", pb.name)
	}
	pod.Spec.NodeSelector = nodeSelector

	runContainer := &pod.Spec.Containers[0]

	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  config.DriverPortEnv,
		Value: fmt.Sprint(config.DriverPort)})

	if xdsServer := kubehelpers.ContainerForName(config.XdsServerContainerName, pod.Spec.Containers); xdsServer != nil {
		if sidecar := kubehelpers.ContainerForName(config.SidecarContainerName, pod.Spec.Containers); sidecar == nil {
			pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: "grpc-xds-bootstrap"})

			runContainer.VolumeMounts = append(runContainer.VolumeMounts, corev1.VolumeMount{
				Name:      "grpc-xds-bootstrap",
				MountPath: "/bootstrap",
				ReadOnly:  true,
			})
			xdsServer.VolumeMounts = append(xdsServer.VolumeMounts, corev1.VolumeMount{
				Name:      "grpc-xds-bootstrap",
				MountPath: "/bootstrap",
				ReadOnly:  false,
			})
		}
	}

	runContainer.Ports = append(runContainer.Ports, corev1.ContainerPort{
		Name:          "driver",
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: config.DriverPort,
	})

	if client.MetricsPort != 0 {
		runContainer.Ports = append(runContainer.Ports, corev1.ContainerPort{
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: client.MetricsPort,
		})
	}

	return pod, nil
}

// PodForDriver accepts a pointer to a driver and returns a pod for it.
func (pb *PodBuilder) PodForDriver(driver *grpcv1.Driver) (*corev1.Pod, error) {
	pb.name = safeStrUnwrap(driver.Name)
	pb.role = config.DriverRole
	pb.pool = safeStrUnwrap(driver.Pool)
	pb.clone = driver.Clone
	pb.build = driver.Build
	pb.run = driver.Run

	pod := pb.newPod()

	nodeSelector := make(map[string]string)
	if driver.Pool != nil {
		nodeSelector["pool"] = *driver.Pool
	} else if pb.defaults.DefaultPoolLabels != nil && pb.defaults.DefaultPoolLabels.Driver != "" {
		nodeSelector[pb.defaults.DefaultPoolLabels.Driver] = "true"
	} else {
		return nil, errors.Wrapf(errNoPool, "could not determine pool for driver (no explicit value or default)")
	}
	pod.Spec.NodeSelector = nodeSelector

	runContainer := &pod.Spec.Containers[0]
	addReadyInitContainer(pb.defaults, pb.test, &pod.Spec, runContainer)

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "scenarios",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: pb.test.Name,
				},
			},
		},
	})
	runContainer.VolumeMounts = append(runContainer.VolumeMounts, corev1.VolumeMount{
		Name:      "scenarios",
		MountPath: config.ScenariosMountPath,
		ReadOnly:  true,
	})
	runContainer.Env = append(runContainer.Env,
		corev1.EnvVar{
			Name:  config.ScenariosFileEnv,
			Value: config.ScenariosMountPath + "/scenarios.json"},
		corev1.EnvVar{
			Name:  "METADATA_OUTPUT_FILE",
			Value: config.ReadyMetadataOutputFile,
		},
		corev1.EnvVar{
			Name:  "NODE_INFO_OUTPUT_FILE",
			Value: config.ReadyNodeInfoOutputFile,
		})

	if results := pb.test.Spec.Results; results != nil {
		if bigQueryTable := results.BigQueryTable; bigQueryTable != nil {
			runContainer.Env = append(runContainer.Env, corev1.EnvVar{
				Name:  config.BigQueryTableEnv,
				Value: *bigQueryTable,
			})
		}
	}

	enablePrometheus, ok := pb.test.Annotations["enablePrometheus"]
	if ok && strings.ToLower(enablePrometheus) == "true" {
		runContainer.Env = append(runContainer.Env,
			corev1.EnvVar{
				Name:  config.EnablePrometheusEnv,
				Value: "true"})
	}

	return pod, nil
}

// PodForServer accepts a pointer to a server and returns a pod for it.
func (pb *PodBuilder) PodForServer(server *grpcv1.Server) (*corev1.Pod, error) {
	pb.name = safeStrUnwrap(server.Name)
	pb.role = config.ServerRole
	pb.pool = safeStrUnwrap(server.Pool)
	pb.clone = server.Clone
	pb.build = server.Build
	pb.run = server.Run

	pod := pb.newPod()

	nodeSelector := make(map[string]string)
	if server.Pool != nil {
		nodeSelector["pool"] = *server.Pool
	} else if pb.defaults.DefaultPoolLabels != nil && pb.defaults.DefaultPoolLabels.Server != "" {
		nodeSelector[pb.defaults.DefaultPoolLabels.Server] = "true"
	} else {
		return nil, errors.Wrapf(errNoPool, "could not determine pool for server %q (no explicit value or default)", pb.name)
	}
	pod.Spec.NodeSelector = nodeSelector

	runContainer := &pod.Spec.Containers[0]

	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  config.DriverPortEnv,
		Value: fmt.Sprintf("%d", config.DriverPort)})

	runContainer.Ports = append(runContainer.Ports, corev1.ContainerPort{
		Name:          "driver",
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: config.DriverPort,
	})

	if server.MetricsPort != 0 {
		runContainer.Ports = append(runContainer.Ports, corev1.ContainerPort{
			Name:          "metrics",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: server.MetricsPort,
		})
	}

	return pod, nil
}

// newPod creates a base pod for any client, driver or server. It is designed to
// be decorated by more specific methods for each of these.
func (pb *PodBuilder) newPod() *corev1.Pod {
	var initContainers []corev1.Container

	if pb.clone != nil {
		var env []corev1.EnvVar

		if pb.clone.Repo != nil {
			env = append(env, corev1.EnvVar{
				Name:  config.CloneRepoEnv,
				Value: safeStrUnwrap(pb.clone.Repo),
			})
		}

		if pb.clone.GitRef != nil {
			env = append(env, corev1.EnvVar{
				Name:  config.CloneGitRefEnv,
				Value: safeStrUnwrap(pb.clone.GitRef),
			})
		}

		initContainers = append(initContainers, corev1.Container{
			Name:  config.CloneInitContainerName,
			Image: safeStrUnwrap(pb.clone.Image),
			Env:   env,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
					ReadOnly:  false,
				},
			},
		})
	}

	if pb.build != nil {
		initContainers = append(initContainers, corev1.Container{
			Name:       config.BuildInitContainerName,
			Image:      safeStrUnwrap(pb.build.Image),
			Command:    pb.build.Command,
			Args:       pb.build.Args,
			Env:        pb.build.Env,
			WorkingDir: config.WorkspaceMountPath,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
					ReadOnly:  false,
				},
				{
					Name:      config.BazelCacheVolumeName,
					MountPath: config.BazelCacheMountPath,
					ReadOnly:  false,
				},
			},
		})
	}

	var runContainers []corev1.Container
	for i, r := range pb.run {
		if i == 0 {
			r.WorkingDir = config.WorkspaceMountPath
			r.VolumeMounts = append(r.VolumeMounts, []corev1.VolumeMount{
				{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
					ReadOnly:  false,
				},
				{
					Name:      config.BazelCacheVolumeName,
					MountPath: config.BazelCacheMountPath,
					ReadOnly:  false,
				}}...)
		}

		if len(r.Env) == 0 {
			r.Env = []corev1.EnvVar{}
		}
		r.Env = append(r.Env, []corev1.EnvVar{
			{
				Name:  config.KillAfterEnv,
				Value: fmt.Sprintf("%f", pb.defaults.KillAfter),
			},
			{
				Name:  config.PodTimeoutEnv,
				Value: fmt.Sprintf("%d", pb.test.Spec.TimeoutSeconds),
			},
		}...)
		runContainers = append(runContainers, r)
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", pb.test.Name, pb.role, pb.name),
			Namespace: pb.test.Namespace,
			Labels: map[string]string{
				config.RoleLabel:          pb.role,
				config.ComponentNameLabel: pb.name,
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: initContainers,
			Containers:     runContainers,
			RestartPolicy:  corev1.RestartPolicyNever,
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      config.RoleLabel,
										Operator: metav1.LabelSelectorOpExists,
									},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: config.WorkspaceVolumeName,
				},
				{
					Name: config.BazelCacheVolumeName,
				},
			},
		},
	}
}

// safeStrUnwrap accepts a string pointer, returning the dereferenced string or
// an empty string if the pointer is nil.
func safeStrUnwrap(strPtr *string) string {
	if strPtr == nil {
		return ""
	}

	return *strPtr
}
