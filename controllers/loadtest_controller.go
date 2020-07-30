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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/pkg/defaults"
)

const reconcileTimeout = 1 * time.Minute

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
