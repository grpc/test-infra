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
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

const (
	podOwnerKey = ".metadata.controller"
)

type LoadTestSet struct {
	tests map[string]bool
}

func (lts *LoadTestSet) initialize() {
	if lts.tests == nil {
		lts.tests = make(map[string]bool)
	}
}

func (lts *LoadTestSet) Add(name string) {
	lts.initialize()
	lts.tests[name] = true
}

func (lts *LoadTestSet) Includes(loadtest grpcv1.LoadTest) bool {
	lts.initialize()
	_, ok := lts.tests[loadtest.Name]
	return ok
}

func getPendingTests(podList corev1.PodList) *LoadTestSet {
	pendingTestNames := &LoadTestSet{}
	for _, pod := range podList.Items {
		if name, ok := pod.Labels[LoadTestLabel]; ok {
			pendingTestNames.Add(name)
		}
	}
	return pendingTestNames
}

// LoadTestReconciler reconciles a LoadTest object
type LoadTestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get

func (r *LoadTestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log := r.Log.WithValues("loadtest", req.NamespacedName)
	poolManager := &PoolManager{}

	var nodes corev1.NodeList
	if err := r.List(ctx, &nodes); err != nil {
		log.Error(err, "failed to list nodes")
		return ctrl.Result{}, err
	}
	poolManager.AddNodeList(nodes)

	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "failed to list pods")
		return ctrl.Result{}, err
	}

	var loadtests grpcv1.LoadTestList
	if err := r.List(ctx, &loadtests); err != nil {
		log.Error(err, "failed to list loadtests")
		return ctrl.Result{}, err
	}

	pendingTests := getPendingTests(pods)
	if err := poolManager.UpdateAvailability(loadtests, pendingTests); err != nil {
		// cluster in bad state!
	}

	var loadtest grpcv1.LoadTest
	if err := r.Get(ctx, req.NamespacedName, &loadtest); err != nil {
		log.Error(err, "failed to get loadtest, it may have been deleted")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Generally, checking a previous state is discouraged for controllers.
	// This controller currently relies on the previous state to determine if a
	// test has terminated, because the pods were likely garbage collected. If
	// the status were to be lost, the loadtest would simply restart.
	lastState := loadtest.Status.State
	if lastState == grpcv1.FailState || lastState == grpcv1.ErrorState || lastState == grpcv1.SuccessState {
		// everything has terminated, so do nothing
		return ctrl.Result{}, nil
	}

	// when no pods are running on the cluster, try to schedule some
	if !pendingTests.Includes(loadtest) {
		var err error

		now := metav1.Now()
		fits, err := poolManager.Fits(loadtest)

		if err != nil {
			loadtest.Status.State = grpcv1.ErrorState
			loadtest.Status.TerminateTime = &now
			log.Error(err, "unable to determine if LoadTest fits with current node availability")

			if updateErr := r.Status().Update(ctx, &loadtest); updateErr != nil {
				log.Error(updateErr, "failed to update loadtest status")
			}

			return ctrl.Result{}, err
		}

		if !fits {
			loadtest.Status.State = grpcv1.WaitingState
			loadtest.Status.AcknowledgeTime = &now
			log.Info("unable to provision due to machine availability, waiting")

			if updateErr := r.Status().Update(ctx, &loadtest); updateErr != nil {
				log.Error(updateErr, "failed to update loadtest status when nodes were unavailable")
			}

			return ctrl.Result{}, nil
		}
	}

	var driverPods []*corev1.Pod
	var serverPods []*corev1.Pod
	var clientPods []*corev1.Pod
	for _, pod := range pods.Items {
		name, ok := pod.Labels[LoadTestLabel]
		if !ok || name != loadtest.Name {
			continue
		}

		role, ok := pod.Labels[RoleLabel]
		if !ok {
			continue
		}

		switch role {
		case DriverRole:
			driverPods = append(driverPods, &pod)
		case ServerRole:
			serverPods = append(serverPods, &pod)
		case ClientRole:
			clientPods = append(clientPods, &pod)
		default:
			continue
		}
	}

	provisioning := false

	if driverPods == nil {
		// create driver and set ProvisioningState
		provisioning = true
		pod := makePod(&loadtest, &loadtest.Spec.Driver.Component, DriverRole, 1)
		if err := ctrl.SetControllerReference(&loadtest, pod, r.Scheme); err != nil {
			log.Error(err, "could not set controller reference on driver")
		}
		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "could not create driver pod")
			return ctrl.Result{}, err
		}
		driverPods = append(driverPods, pod)
	} else if len(driverPods) > 1 {
		// delete a driver
	}

	if len(serverPods) < len(loadtest.Spec.Servers) {
		// create server pod and set ProvisioningState
		provisioning = true
		pod := makePod(&loadtest, &loadtest.Spec.Servers[0].Component, ServerRole, 1)
		if err := ctrl.SetControllerReference(&loadtest, pod, r.Scheme); err != nil {
			log.Error(err, "could not set controller reference on server")
		}
		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "could not create server pod")
			return ctrl.Result{}, err
		}
		serverPods = append(serverPods, pod)
	} else if len(serverPods) > len(loadtest.Spec.Servers) {
		// delete a server
	}

	if len(clientPods) < len(loadtest.Spec.Clients) {
		// create client pod and set ProvisioningState
		provisioning = true
		pod := makePod(&loadtest, &loadtest.Spec.Clients[0].Component, ClientRole, 1)
		if err := ctrl.SetControllerReference(&loadtest, pod, r.Scheme); err != nil {
			log.Error(err, "could not set controller reference on client")
		}
		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "could not create client pod")
			return ctrl.Result{}, err
		}
		clientPods = append(clientPods, pod)
	} else if len(clientPods) > len(loadtest.Spec.Clients) {
		// delete a client
	}

	if provisioning {
		loadtest.Status.State = grpcv1.ProvisioningState
		if loadtest.Status.ProvisionTime == nil {
			now := metav1.Now()
			loadtest.Status.ProvisionTime = &now
		}

		if updateErr := r.Status().Update(ctx, &loadtest); updateErr != nil {
			log.Error(updateErr, "failed to update loadtest status when provisioning")
		}
	}

	var testPods []*corev1.Pod
	testPods = append(testPods, driverPods...)
	testPods = append(testPods, serverPods...)
	testPods = append(testPods, clientPods...)

	// check the status of pods and update status
	var badPod *corev1.Pod
	var state grpcv1.LoadTestState = grpcv1.PendingState
	now := metav1.Now()

	for _, pod := range testPods {
		status := &pod.Status

		if count := len(status.ContainerStatuses); count != 1 {
			continue
		}
		containerStatus := status.ContainerStatuses[0]

		terminationState := containerStatus.LastTerminationState.Terminated
		if terminationState == nil {
			terminationState = containerStatus.State.Terminated
		}

		if terminationState != nil {
			loadtest.Status.TerminateTime = &now

			if terminationState.ExitCode == 0 {
				state = grpcv1.SuccessState
			} else {
				state = grpcv1.FailState
				badPod = pod
				break
			}
		}

		if waitingState := containerStatus.State.Waiting; waitingState != nil {
			loadtest.Status.TerminateTime = &now

			if strings.Compare("CrashLoopBackOff", waitingState.Reason) == 0 {
				state = grpcv1.ErrorState
				badPod = pod
				break
			}
		}
	}

	loadtest.Status.State = state
	if updateErr := r.Status().Update(ctx, &loadtest); updateErr != nil {
		log.Error(updateErr, "failed to update loadtest status", "state", state, "offendingPod", badPod)
	}

	return ctrl.Result{}, nil
}

func (r *LoadTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	podOwnerIndexFunc := func(obj runtime.Object) []string {
		pod := obj.(*corev1.Pod)
		controller := metav1.GetControllerOf(pod)
		if controller == nil || controller.Kind != "LoadTest" {
			return nil
		}
		return []string{controller.Name}
	}

	if err := mgr.GetFieldIndexer().IndexField(&corev1.Pod{}, podOwnerKey, podOwnerIndexFunc); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// type NodePool struct {
// 	Nodes []*corev1.Node
// 	Available int
// 	Capacity int
// }

// type ClusterGraph struct {
// 	nodes []*corev1.Node
// 	pools []*NodePool
// 	pods []*corev1.Pod
// 	tests []*grpcv1.LoadTest
// }

// func (c *ClusterGraph) AddNodes(nodes []corev1.Node) {

// }
