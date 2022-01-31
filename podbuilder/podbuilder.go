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
	"log"
	"strconv"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/kubehelpers"
)

// errNoPool is the base error when a PodBuilder cannot determine the pool for
// a pod.
var errNoPool = errors.New("pool is missing")

// errTestType is the vase error when a PodBuilder cannot determine the type of
// the test.
var errTestType = errors.New("failed to determine the test type")

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
			{
				Name:  config.PSMTestServerPortEnv,
				Value: defs.PSMTestServerPort,
			},
			{
				Name:  config.XDSEndpointUpdatePortEnv,
				Value: defs.XDSEndpointUpdatePort,
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
	run      *grpcv1.Run
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
	pb.run = &client.Run

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

	runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)

	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  config.DriverPortEnv,
		Value: fmt.Sprint(config.DriverPort)})

	runContainer.Ports = append(runContainer.Ports, corev1.ContainerPort{
		Name:          "driver",
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: config.DriverPort,
	})

	// sidecar and xds server will keep the client pod alive even after the test
	// finishes. A livenessPorbe is configured to check on worker's driver port,
	// one the driver port on workers is closed the sidecar and xds containers
	// become unhealthy and kill themsleves.

	initialDelaySeconds := int32(30)
	periodSeconds := int32(5)

	if initialDelaySecondsValue, ok := pb.test.Annotations["initialDelaySeconds"]; ok {
		initialDelaySeconds64, err := strconv.ParseInt(initialDelaySecondsValue, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse the initial delay seconds for sidecar/xds containers's liveness probe: %v", initialDelaySecondsValue)
		}
		initialDelaySeconds = int32(initialDelaySeconds64)
	}

	if periodSecondsValue, ok := pb.test.Annotations["periodSeconds"]; ok {
		periodSeconds64, err := strconv.ParseInt(periodSecondsValue, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse the polling seconds for sidecar/xds containers's liveness probe: %v", periodSecondsValue)
		}
		periodSeconds = int32(periodSeconds64)
	}

	if client.XDS != nil {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: config.NonProxiedBootstrapVolumeName,
		})

		runContainer.Env = append(runContainer.Env, corev1.EnvVar{
			Name:  "GRPC_XDS_BOOTSTRAP",
			Value: config.NonProxiedBootstrapMountPath + "/bootstrap.json"})
		runContainer.VolumeMounts = append(runContainer.VolumeMounts,
			corev1.VolumeMount{
				Name:      config.NonProxiedBootstrapVolumeName,
				MountPath: config.NonProxiedBootstrapMountPath,
				ReadOnly:  false})

		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
			Name:    config.XDSServerContainerName,
			Image:   safeStrUnwrap(client.XDS.Image),
			Command: client.XDS.Command,
			Args:    client.XDS.Args,
			Env: []corev1.EnvVar{
				{
					Name:  config.KillAfterEnv,
					Value: fmt.Sprintf("%f", pb.defaults.KillAfter),
				},
				{
					Name:  config.PodTimeoutEnv,
					Value: fmt.Sprintf("%d", pb.test.Spec.TimeoutSeconds),
				},
				{
					Name:  config.NonProxiedTargetStringEnv,
					Value: pb.defaults.NonProxiedTargetString,
				},
				{
					Name:  config.SidecarListenerPortEnv,
					Value: pb.defaults.SidecarListenerPort,
				},
			},
			WorkingDir: config.WorkspaceMountPath,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      config.NonProxiedBootstrapVolumeName,
					MountPath: config.NonProxiedBootstrapMountPath,
					ReadOnly:  false,
				},
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							IntVal: config.DriverPort},
						Host: "localhost",
					},
				},
				FailureThreshold:    1,
				InitialDelaySeconds: initialDelaySeconds,
				PeriodSeconds:       periodSeconds,
			},
		})

		if client.Sidecar != nil {
			log.Print("running test with sidecar")
			pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
				Name:    config.SidecarContainerName,
				Image:   safeStrUnwrap(client.Sidecar.Image),
				Command: client.Sidecar.Command,
				Args:    client.Sidecar.Args,
				LivenessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.IntOrString{
								IntVal: config.DriverPort},
							Host: "localhost",
						},
					},
					FailureThreshold:    1,
					InitialDelaySeconds: initialDelaySeconds,
					PeriodSeconds:       periodSeconds,
				},
			})
		} else {
			log.Print("running gRPC proxyless service mesh test")
		}
	} else {
		log.Print("No sidecar or xDS server images provided, running regular load test")
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
	pb.run = &driver.Run

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

	runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
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

	isPSMTest, err := kubehelpers.IsPSMTest(&pb.test.Spec.Clients)
	if err != nil {
		return nil, errors.Wrapf(errTestType, "could not determine test type for test %q: %v", pb.test.Name, err)
	}
	isProxiedTest, err := kubehelpers.IsProxiedTest(&pb.test.Spec.Clients)
	if err != nil {
		return nil, errors.Wrapf(errTestType, "could not determine test type for test %q: %v", pb.test.Name, err)
	}

	serverTargetStringOverride := ""

	if isPSMTest {
		if isProxiedTest {
			serverTargetStringOverride = fmt.Sprintf("localhost:%v", pb.defaults.SidecarListenerPort)
		} else {
			serverTargetStringOverride = fmt.Sprintf("xds:///%v", pb.defaults.NonProxiedTargetString)
		}
	}
	runContainer.Env = append(runContainer.Env,
		corev1.EnvVar{
			Name:  config.TargetStringOverrideEnv,
			Value: serverTargetStringOverride,
		})

	if results := pb.test.Spec.Results; results != nil {
		if bigQueryTable := results.BigQueryTable; bigQueryTable != nil {
			runContainer.Env = append(runContainer.Env, corev1.EnvVar{
				Name:  config.BigQueryTableEnv,
				Value: *bigQueryTable,
			})
		}
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
	pb.run = &server.Run

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

	runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)

	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  config.DriverPortEnv,
		Value: fmt.Sprintf("%d", config.DriverPort)})

	runContainer.Ports = append(runContainer.Ports, corev1.ContainerPort{
		Name:          "driver",
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: config.DriverPort,
	})

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
			Containers: []corev1.Container{
				{
					Name:    config.RunContainerName,
					Image:   safeStrUnwrap(pb.run.Image),
					Command: pb.run.Command,
					Args:    pb.run.Args,
					Env: []corev1.EnvVar{
						{
							Name:  config.KillAfterEnv,
							Value: fmt.Sprintf("%f", pb.defaults.KillAfter),
						},
						{
							Name:  config.PodTimeoutEnv,
							Value: fmt.Sprintf("%d", pb.test.Spec.TimeoutSeconds),
						},
						{
							Name:  "GRPC_GO_LOG_VERBOSITY_LEVEL",
							Value: "99",
						},
						{
							Name:  "GRPC_GO_LOG_SEVERITY_LEVEL",
							Value: "info",
						},
					},
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
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
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
