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

package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/status"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/proto/grpc/testing"
)

// CleanupAgent

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get

// Agent cleanup unwanted processes.
type Agent struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile attempts to check status of workers of the triggering LoadTest, if
// a terminated LoadTest has workers still running, reconcile will send quit RPC
// to stop the workers.
func (c *Agent) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	var err error

	log := c.Log.WithValues("loadtest", req.NamespacedName)

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// Fetch the LoadTest that triggers the event.
	loadtest := new(grpcv1.LoadTest)
	if err = c.Get(ctx, req.NamespacedName, loadtest); err != nil {
		log.Error(err, "failed to get LoadTest", "name", req.NamespacedName)
		// do not requeue, the load test may have been deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info(loadtest.Status.Message)
	log.Info(loadtest.Status.Reason)

	// If the triggering LoadTest is not yet terminated do nothing
	if !loadtest.Status.State.IsTerminated() {
		return ctrl.Result{}, nil
	}

	// Fetch all the pods live on the cluster.
	pods := new(corev1.PodList)
	if err = c.List(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "failed to list pods", "namespace", req.Namespace)
		return ctrl.Result{Requeue: true}, err
	}

	// Reuse existing logic to find the pods related to triggering LoadTest.
	ownedPods := status.PodsForLoadTest(loadtest, pods.Items)

	// Check if any related pods are needed to be terminated and attempt to
	// terminate them.
	for _, pod := range ownedPods {

		if pod.Labels[config.RoleLabel] == config.DriverRole {
			continue
		}

		if pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded {
			callQuit(pod, c.Log, req.NamespacedName)
		}
	}
	return ctrl.Result{}, nil

}

// callQuit attempts to send quit RPC to the given pod. It takes reference of the
// to be stopped pod, start a connection and send quit RPC to that pod. The
// connection and signal sending are wrapped in the same context limiting total
// time spend on each pod.
func callQuit(currentPod *corev1.Pod, log logr.Logger, namespacedName types.NamespacedName) {
	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Info(fmt.Sprintf("the current address and port is : %v", currentPod.Status.PodIP+":10000"))

	conn, err := grpc.DialContext(ctx, currentPod.Status.PodIP+":10000", grpc.WithInsecure())
	if err != nil {
		log.Error(err, "failed to build the connection", "name", namespacedName)
	}

	currentClient := grpc_testing.NewWorkerServiceClient(conn)

	_, err = currentClient.QuitWorker(ctx, &grpc_testing.Void{}, grpc.WaitForReady(false))
	if err != nil {
		log.Error(err, "failed to send quit RPC", "name", namespacedName)
	}
	conn.Close()
}

// SetupWithManager configures a controller-runtime manager.
func (c *Agent) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Complete(c)
}
