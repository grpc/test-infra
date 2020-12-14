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

// Package cleanup contains logic of cleanup agent. A cleanup agent watches the
// status change of LoadTests. The cleanup agent do nothing when the
// triggering LoadTest is not terminated. When triggering LoadTest is
// terminated, the cleanup agent checks if all its pods are terminated, and send
// quit RPC to stop the those unterminated.
package cleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	pb "github.com/grpc/test-infra/proto/grpc/testing"
	"github.com/grpc/test-infra/status"
)

// Agent cleanup unwanted processes.
type Agent struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Timeout time.Duration
}

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get

// Reconcile attempts to check status of workers of the triggering LoadTest. If
// a terminated LoadTest has workers running, reconcile will send callQuitter RPC
// to stop the workers.
func (a *Agent) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	var err error

	// A timeout could be set on cleanup agent as a time limit for each reconcile
	// round, if not set the default value for each reconciliation is 2 mins.
	if a.Timeout == 0 {
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), a.Timeout)
	}
	defer cancel()

	log := a.Log.WithValues("loadtest", req.NamespacedName)

	// Fetch the LoadTest that triggers the event.
	loadtest := new(grpcv1.LoadTest)
	if err = a.Get(ctx, req.NamespacedName, loadtest); err != nil {
		log.Error(err, "failed to get LoadTest")
		// do not requeue, the load test may have been deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If the triggering LoadTest is not yet terminated, do nothing.
	if !loadtest.Status.State.IsTerminated() {
		return ctrl.Result{}, nil
	}

	// Fetch all the pods live on the cluster.
	pods := new(corev1.PodList)
	if err = a.List(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "failed to list pods")
		return ctrl.Result{Requeue: true}, err
	}

	// Reuse existing logic to find the pods related to triggering LoadTest.
	ownedPods := status.PodsForLoadTest(loadtest, pods.Items)

	q := quitClient{}
	quitWorkers(ctx, &q, ownedPods, log)

	return ctrl.Result{}, nil
}

type callQuitter interface {
	callQuit(context.Context, *corev1.Pod, logr.Logger)
}

type quitClient struct {
}

// quitWorkers takes a list of pods and a log, check on each pod and send callQuitter
// RPC if the pod is a worker with status of running, pending and unknown.
func quitWorkers(ctx context.Context, q callQuitter, ownedPods []*corev1.Pod, log logr.Logger) {
	for i := range ownedPods {
		if ownedPods[i].Labels[config.RoleLabel] == config.DriverRole {
			continue
		}

		padStatus, _, _ := status.StateForPodStatus(&ownedPods[i].Status)
		if padStatus == status.Pending {
			q.callQuit(ctx, ownedPods[i], log)
		}
	}
}

// callQuit establishes a connection with a pod and sends callQuitter RPC
// with a time limit. The pod is assumed to implement callQuitter.
func (c *quitClient) callQuit(ctx context.Context, pod *corev1.Pod, log logr.Logger) {

	target := fmt.Sprintf("%s:%d", pod.Status.PodIP, config.DriverPort)
	conn, err := grpc.DialContext(ctx, target, grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		log.Error(err, "failed to connect to pod", "podName", pod.Labels[config.ComponentNameLabel])
		return
	}
	client := pb.NewWorkerServiceClient(conn)

	_, err = client.QuitWorker(ctx, &pb.Void{}, grpc.WaitForReady(false))

	if err != nil {
		log.Error(err, "failed to callQuitter the worker", "podName", pod.Labels[config.ComponentNameLabel])
		return
	}
}

// SetupWithManager configures a controller-runtime manager.
func (a *Agent) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Complete(a)
}
