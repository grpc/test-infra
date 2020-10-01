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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/grpc/test-infra/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
)

// reconcileTimeout specifies the maximum amount of time any set of API
// requests should take for a single invocation of the Reconcile method.
const reconcileTimeout = 1 * time.Minute

// LoadTestReconciler reconciles a LoadTest object
type LoadTestReconciler struct {
	client.Client
	Defaults *config.Defaults
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get

// Reconcile attempts to bring the current state of the load test into agreement
// with its declared spec. This may mean provisioning resources, doing nothing
// or handling the termination of its pods.
func (r *LoadTestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var err error

	log := r.Log.WithValues("loadtest", req.NamespacedName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Fetch the current state of the world.

	// Listing nodes will be required to achieve gang scheduling; however, this is
	// not implemented at this time. Therefore, it is commented here for later:
	//
	//nodes := new(corev1.NodeList)
	//if err = r.List(ctx, nodes); err != nil {
	//	log.Error(err, "failed to list nodes")
	//	return ctrl.Result{Requeue: true}, err
	//}

	pods := new(corev1.PodList)
	if err = r.List(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "failed to list pods", "namespace", req.Namespace)
		return ctrl.Result{Requeue: true}, err
	}

	loadtests := new(grpcv1.LoadTestList)
	if err = r.List(ctx, loadtests); err != nil {
		log.Error(err, "failed to list loadtests")
		return ctrl.Result{Requeue: true}, err
	}

	loadtest := new(grpcv1.LoadTest)
	if err = r.Get(ctx, req.NamespacedName, loadtest); err != nil {
		log.Error(err, "failed to get loadtest", "name", req.NamespacedName)

		// do not requeue, the load test may have been deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if loadtest.Status.State.IsTerminated() {
		// the load test has already completed, this is probably garbage collection
		return ctrl.Result{}, nil
	}

	loadtest = loadtest.DeepCopy()
	if err = r.Defaults.SetLoadTestDefaults(loadtest); err != nil {
		log.Error(err, "failed to set defaults on loadtest")

		// do not requeue, something has gone horribly wrong
		return ctrl.Result{}, err
	}
	if err = r.Update(ctx, loadtest); err != nil {
		log.Error(err, "failed to update loadtest with defaults")
		return ctrl.Result{Requeue: true}, err
	}

	ownedPods := status.PodsForLoadTest(loadtest, pods.Items)
	loadtest.Status = status.ForLoadTest(loadtest, ownedPods)

	if err = r.Status().Update(ctx, loadtest); err != nil {
		log.Error(err, "failed to update loadtest status")
		return ctrl.Result{Requeue: true}, err
	}

	var pod *corev1.Pod
	missingPods := status.CheckMissingPods(loadtest, ownedPods)

	if len(missingPods.Servers) > 0 {
		pod, err = newServerPod(r.Defaults, loadtest, &missingPods.Servers[0].Component)
	} else if len(missingPods.Clients) > 0 {
		pod, err = newClientPod(r.Defaults, loadtest, &missingPods.Clients[0].Component)
	} else if missingPods.Driver != nil {
		pod, err = newDriverPod(r.Defaults, loadtest, &missingPods.Driver.Component)
	}

	if err != nil {
		log.Error(err, "could not initialize new pod", "pod", pod)
	}
	if pod != nil {
		if err = ctrl.SetControllerReference(loadtest, pod, r.Scheme); err != nil {
			log.Error(err, "could not set controller reference on pod", "pod", pod)
			return ctrl.Result{Requeue: true}, err
		}

		if err = r.Create(ctx, pod); err != nil {
			log.Error(err, "could not create new pod", "pod", pod)
			return ctrl.Result{Requeue: true}, err
		}
	}

	// TODO: Add logic to schedule the next missing pod.

	// PLACEHOLDERS!
	_ = pods
	_ = loadtests
	_ = loadtest
	return ctrl.Result{}, nil
}

// LoadTestMissing categorize missing components based on their roles at specific
// moment. The struct is a wrapper to help us get role information associate
// with components.

// SetupWithManager configures a controller-runtime manager.
func (r *LoadTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// newClientPod creates a client given defaults, a load test and a reference to
// the client's component. It returns an error if a pod cannot be constructed.
func newClientPod(defs *config.Defaults, loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, config.ClientRole)
	if err != nil {
		return nil, err
	}

	addDriverPort(&pod.Spec.Containers[0], defs.DriverPort)

	return pod, nil
}

// scenarioVolumeName accepts the name of a ConfigMap with a scenario and
// generates a name for a volume.
func scenarioVolumeName(scenario string) string {
	return fmt.Sprintf("scenario-%s", scenario)
}

// newScenarioVolume accepts the name of a scenario ConfigMap and returns a
// volume that can mount it.
func newScenarioVolume(scenario string) corev1.Volume {
	return corev1.Volume{
		Name: scenarioVolumeName(scenario),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: scenario,
				},
			},
		},
	}
}

// newScenarioVolumeMount accepts the name of a scenario ConfigMap and returns a
// volume mount that will place it at /src/scenarios.
func newScenarioVolumeMount(scenario string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      scenarioVolumeName(scenario),
		MountPath: config.ScenariosMountPath,
		ReadOnly:  true,
	}
}

// newWorkspaceVolume returns an emptyDir volume with
// `config.WorkspaceVolumeName` as its name.
func newWorkspaceVolume() corev1.Volume {
	return corev1.Volume{Name: config.WorkspaceVolumeName}
}

// newWorkspaceVolumeMount returns a volume mount with
// `config.WorkspaceMountPath` as the path and a reference to the
// `config.WorkspaceVolumeName`. This volume mount grants read/write access.
func newWorkspaceVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      config.WorkspaceVolumeName,
		MountPath: config.WorkspaceMountPath,
		ReadOnly:  false,
	}
}

// newScenarioEnvVar accepts the name of a scenario ConfigMap and returns a
// an environment variable used to locate the scenario in a mounted volume.
func newScenarioFileEnvVar(scenario string) corev1.EnvVar {
	scenarioFile := strings.ReplaceAll(scenario, "-", "_") + ".json"
	return corev1.EnvVar{
		Name:  config.ScenariosFileEnv,
		Value: config.ScenariosMountPath + "/" + scenarioFile,
	}
}

// addReadyInitContainer configures a ready init container. This container is
// meant to wait for workers to become ready, writing the IP address and port of
// these workers to a file. This file is then shared over a volume with the
// driver's run container.
//
// This method also sets the $QPS_WORKERS_FILE environment variable on the
// driver's run container. Its value will point to the aforementioned, shared
// file.
func addReadyInitContainer(defs *config.Defaults, loadtest *grpcv1.LoadTest, podspec *corev1.PodSpec, container *corev1.Container) {
	if defs == nil || podspec == nil || container == nil {
		return
	}

	readyCont := newReadyContainer(defs, loadtest)
	podspec.InitContainers = append(podspec.InitContainers, readyCont)

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

// newBigQueryTableEnvVar accepts a table name and returns the environment
// variable that must be set on the driver to write to the table.
func newBigQueryTableEnvVar(tableName string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  config.BigQueryTableEnv,
		Value: tableName,
	}
}

// newDriverPod creates a driver given defaults, a load test and a reference to
// the driver's component. It returns an error if a pod cannot be constructed.
func newDriverPod(defs *config.Defaults, loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, config.DriverRole)
	if err != nil {
		return nil, err
	}

	podSpec := &pod.Spec
	testSpec := &loadtest.Spec

	// TODO: Avoid referencing containers by index, use names
	testContainer := &podSpec.Containers[0]
	addReadyInitContainer(defs, loadtest, podSpec, testContainer)

	// TODO: Handle more than 1 scenario
	if len(testSpec.Scenarios) > 0 {
		scenario := testSpec.Scenarios[0].Name
		podSpec.Volumes = append(podSpec.Volumes, newScenarioVolume(scenario))
		testContainer.VolumeMounts = append(testContainer.VolumeMounts, newScenarioVolumeMount(scenario))
		testContainer.Env = append(testContainer.Env, newScenarioFileEnvVar(scenario))
	}

	if results := testSpec.Results; results != nil {
		if bigQueryTable := results.BigQueryTable; bigQueryTable != nil {
			testContainer.Env = append(testContainer.Env, newBigQueryTableEnvVar(*bigQueryTable))
		}
	}

	return pod, nil
}

// addDriverPort decorates a container with an additional port for the driver
// and `--driver_port` flag set to its number.
func addDriverPort(container *corev1.Container, portNumber int32) {
	container.Ports = append(container.Ports, newContainerPort("driver", portNumber))
	container.Args = append(container.Args, fmt.Sprintf("--driver_port=%d", portNumber))
}

// addServerPort decorates a container with an additional port for the server
// and `--server_port` flag set to its number.
func addServerPort(container *corev1.Container, portNumber int32) {
	container.Ports = append(container.Ports, newContainerPort("server", portNumber))
	container.Args = append(container.Args, fmt.Sprintf("--server_port=%d", portNumber))
}

// newContainerPort creates a Kubernetes ContainerPort object with the provided
// name and portNumber. The name should uniquely identify the port and the port
// number must be within the standard port range. The protocol is assumed to be
// TCP.
func newContainerPort(name string, portNumber int32) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          name,
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: portNumber,
	}
}

// newServerPod creates a server given defaults, a load test and a reference to
// the server's component. It returns an error if a pod cannot be constructed.
func newServerPod(defs *config.Defaults, loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, config.ServerRole)
	if err != nil {
		return nil, err
	}

	rc := &pod.Spec.Containers[0]
	addDriverPort(rc, defs.DriverPort)
	addServerPort(rc, defs.ServerPort)

	return pod, nil
}

// newCloneContainer constructs a container given a grpcv1.Clone pointer. If
// the pointer is nil, an empty container is returned.
func newCloneContainer(clone *grpcv1.Clone) corev1.Container {
	if clone == nil {
		return corev1.Container{}
	}

	var env []corev1.EnvVar

	if clone.Repo != nil {
		env = append(env, corev1.EnvVar{Name: config.CloneRepoEnv, Value: *clone.Repo})
	}

	if clone.GitRef != nil {
		env = append(env, corev1.EnvVar{Name: config.CloneGitRefEnv, Value: *clone.GitRef})
	}

	return corev1.Container{
		Name:  config.CloneInitContainerName,
		Image: safeStrUnwrap(clone.Image),
		Env:   env,
		VolumeMounts: []corev1.VolumeMount{
			newWorkspaceVolumeMount(),
		},
	}
}

// newBuildContainer constructs a container given a grpcv1.Build pointer. If
// the pointer is nil, an empty container is returned.
func newBuildContainer(build *grpcv1.Build) corev1.Container {
	if build == nil {
		return corev1.Container{}
	}

	return corev1.Container{
		Name:       config.BuildInitContainerName,
		Image:      *build.Image,
		Command:    build.Command,
		Args:       build.Args,
		Env:        build.Env,
		WorkingDir: config.WorkspaceMountPath,
		VolumeMounts: []corev1.VolumeMount{
			newWorkspaceVolumeMount(),
		},
	}
}

// newReadyContainer constructs a container using the default ready container
// image. If defaults parameter is nil, an empty container is returned.
func newReadyContainer(defs *config.Defaults, loadtest *grpcv1.LoadTest) corev1.Container {
	if defs == nil {
		return corev1.Container{}
	}

	var args []string
	for _, server := range loadtest.Spec.Servers {
		args = append(args, fmt.Sprintf("%s=%s,%s=%s,%s=%s",
			config.LoadTestLabel, loadtest.Name,
			config.RoleLabel, config.ServerRole,
			config.ComponentNameLabel, *server.Name,
		))
	}
	for _, client := range loadtest.Spec.Clients {
		args = append(args, fmt.Sprintf("%s=%s,%s=%s,%s=%s",
			config.LoadTestLabel, loadtest.Name,
			config.RoleLabel, config.ClientRole,
			config.ComponentNameLabel, *client.Name,
		))
	}

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
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      config.ReadyVolumeName,
				MountPath: config.ReadyMountPath,
			},
		},
	}
}

// newRunContainer constructs a container given a grpcv1.Run object.
func newRunContainer(run grpcv1.Run) corev1.Container {
	return corev1.Container{
		Name:       config.RunContainerName,
		Image:      *run.Image,
		Command:    run.Command,
		Args:       run.Args,
		Env:        run.Env,
		WorkingDir: config.WorkspaceMountPath,
		VolumeMounts: []corev1.VolumeMount{
			newWorkspaceVolumeMount(),
		},
	}
}

// newPod constructs a Kubernetes pod.
func newPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component, role string) (*corev1.Pod, error) {
	var initContainers []corev1.Container

	if component.Clone != nil {
		initContainers = append(initContainers, newCloneContainer(component.Clone))
	}

	if component.Build != nil {
		initContainers = append(initContainers, newBuildContainer(component.Build))
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", loadtest.Name, role, *component.Name),
			Namespace: loadtest.Namespace,
			Labels: map[string]string{
				config.LoadTestLabel:      loadtest.Name,
				config.RoleLabel:          role,
				config.ComponentNameLabel: *component.Name,
			},
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"pool": *component.Pool,
			},
			InitContainers: initContainers,
			Containers:     []corev1.Container{newRunContainer(component.Run)},
			RestartPolicy:  corev1.RestartPolicyNever,
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      config.LoadTestLabel,
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
				newWorkspaceVolume(),
			},
		},
	}, nil
}

// safeStrUnwrap accepts a string pointer, returning the dereferenced string or
// an empty string if the pointer is nil.
func safeStrUnwrap(strPtr *string) string {
	if strPtr == nil {
		return ""
	}

	return *strPtr
}
