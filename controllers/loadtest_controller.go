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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/pkg/defaults"
)

// reconcileTimeout specifies the maximum amount of time any set of API
// requests should take for a single invocation of the Reconcile method.
const reconcileTimeout = 1 * time.Minute

// cloneInitContainer holds the name of the init container that obtains a copy
// of the code at a specific point in time.
const cloneInitContainer = "clone"

// buildInitContainer holds the name of the init container that assembles a
// binary or other bundle required to run the tests.
const buildInitContainer = "build"

// runContainer holds the name of the main container where the test is executed.
const runContainer = "run"

// scenarioMountPath specifies where the JSON file with the scenario should be
// mounted in the driver container.
const scenarioMountPath = "/src/scenarios"

// scenarioFileEnv specifies the name of an env variable that specifies the path
// to a JSON file with a scenario.
const scenarioFileEnv = "SCENARIO_FILE"

// CloneRepoEnv specifies the name of the env variable that contains the git
// repository to clone.
const CloneRepoEnv = "CLONE_REPO"

// CloneGitRefEnv specifies the name of the env variable that contains the
// commit, tag or branch to checkout after cloning a git repository.
const CloneGitRefEnv = "CLONE_GIT_REF"

// LoadTestReconciler reconciles a LoadTest object
type LoadTestReconciler struct {
	client.Client
	Defaults *defaults.Defaults
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch

// Reconcile attempts to bring the current state of the load test into agreement
// with its declared spec. This may mean provisioning resources, doing nothing
// or handling the termination of its pods.
func (r *LoadTestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("loadtest", req.NamespacedName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Fetch the current state of the world.

	var nodes corev1.NodeList
	if err := r.List(ctx, &nodes); err != nil {
		log.Error(err, "failed to list nodes")
		// attempt to requeue with exponential back-off
		return ctrl.Result{Requeue: true}, err
	}

	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "failed to list pods", "namespace", req.Namespace)
		// attempt to requeue with exponential back-off
		return ctrl.Result{Requeue: true}, err
	}

	var loadtests grpcv1.LoadTestList
	if err := r.List(ctx, &loadtests); err != nil {
		log.Error(err, "failed to list loadtests")
		// attempt to requeue with exponential back-off
		return ctrl.Result{Requeue: true}, err
	}

	var loadtest grpcv1.LoadTest
	if err := r.Get(ctx, req.NamespacedName, &loadtest); err != nil {
		log.Error(err, "failed to get loadtest", "name", req.NamespacedName)
		// do not requeue, may have been garbage collected
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the loadtest has terminated.

	// TODO: Do nothing if the loadtest has terminated.

	// Check the status of any running pods.

	// TODO: Add method to get list of owned pods and method to check their status.

	// Create any missing pods that the loadtest needs.

	// TODO: Add method to detect what is missing on the load test.
	// TODO: Add logic to schedule the next missing pod.

	// PLACEHOLDERS!
	_ = nodes
	_ = pods
	_ = loadtests
	_ = loadtest
	return ctrl.Result{}, nil
}

// SetupWithManager configures a controller-runtime manager.
func (r *LoadTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Complete(r)
}

// newClientPod creates a client given a load test and a reference to its
// component. It returns an error if a pod cannot be constructed.
func newClientPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, defaults.ClientRole)
	if err != nil {
		return nil, err
	}

	addDriverPort(&pod.Spec.Containers[0])

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
		MountPath: scenarioMountPath,
		ReadOnly:  true,
	}
}

// newScenarioEnvVar accepts the name of a scenario ConfigMap and returns a
// an environment variable used to locate the scenario in a mounted volume.
func newScenarioFileEnvVar(scenario string) corev1.EnvVar {
	scenarioFile := strings.ReplaceAll(scenario, "-", "_") + ".json"
	return corev1.EnvVar{
		Name:  scenarioFileEnv,
		Value: scenarioMountPath + "/" + scenarioFile,
	}
}

// newDriverPod creates a driver given a load test and a reference to its
// component. It returns an error if a pod cannot be constructed.
func newDriverPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, defaults.DriverRole)
	if err != nil {
		return nil, err
	}

	podSpec := &pod.Spec
	testSpec := &loadtest.Spec

	// TODO: Avoid referencing containers by index, use names
	testContainer := &podSpec.Containers[0]

	addDriverPort(testContainer)

	// TODO: Handle more than 1 scenario
	if len(testSpec.Scenarios) > 0 {
		scenario := testSpec.Scenarios[0].Name
		podSpec.Volumes = append(podSpec.Volumes, newScenarioVolume(scenario))
		testContainer.VolumeMounts = append(testContainer.VolumeMounts, newScenarioVolumeMount(scenario))
		testContainer.Env = append(testContainer.Env, newScenarioFileEnvVar(scenario))
	}

	return pod, nil
}

// addDriverPort decorates a container with an additional port for the driver.
func addDriverPort(container *corev1.Container) {
	container.Ports = append(container.Ports, newContainerPort("driver", 10000))
}

// addServerPort decorates a container with an additional port for the server.
func addServerPort(container *corev1.Container) {
	container.Ports = append(container.Ports, newContainerPort("server", 10010))
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

// newServerPod creates a server given a load test and a reference to its
// component. It returns an error if a pod cannot be constructed.
func newServerPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, defaults.ServerRole)
	if err != nil {
		return nil, err
	}

	addDriverPort(&pod.Spec.Containers[0])
	addServerPort(&pod.Spec.Containers[0])

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
		env = append(env, corev1.EnvVar{Name: CloneRepoEnv, Value: *clone.Repo})
	}

	if clone.GitRef != nil {
		env = append(env, corev1.EnvVar{Name: CloneGitRefEnv, Value: *clone.GitRef})
	}

	return corev1.Container{
		Name:  cloneInitContainer,
		Image: safeStrUnwrap(clone.Image),
		Env:   env,
	}
}

// newBuildContainer constructs a container given a grpcv1.Build pointer. If
// the pointer is nil, an empty container is returned.
func newBuildContainer(build *grpcv1.Build) corev1.Container {
	if build == nil {
		return corev1.Container{}
	}

	return corev1.Container{
		Name:    buildInitContainer,
		Image:   *build.Image,
		Command: build.Command,
		Args:    build.Args,
		Env:     build.Env,
	}
}

// newRunContainer constructs a container given a grpcv1.Run object.
func newRunContainer(run grpcv1.Run) corev1.Container {
	return corev1.Container{
		Name:    runContainer,
		Image:   *run.Image,
		Command: run.Command,
		Args:    run.Args,
		Env:     run.Env,
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
			Name: fmt.Sprintf("%s-%s-%s", loadtest.Name, role, *component.Name),
			Labels: map[string]string{
				defaults.LoadTestLabel:      loadtest.Name,
				defaults.RoleLabel:          role,
				defaults.ComponentNameLabel: *component.Name,
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
