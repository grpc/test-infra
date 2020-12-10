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
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/podbuilder"
	"github.com/grpc/test-infra/status"
)

// LoadTestReconciler reconciles a LoadTest object
type LoadTestReconciler struct {
	client.Client
	Defaults *config.Defaults
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Timeout  time.Duration
}

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile attempts to bring the current state of the load test into agreement
// with its declared spec. This may mean provisioning resources, doing nothing
// or handling the termination of its pods.
func (r *LoadTestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	var err error
	log := r.Log.WithValues("loadtest", req.NamespacedName)

	if r.Timeout == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), r.Timeout)
	}
	defer cancel()

	rawTest := new(grpcv1.LoadTest)
	if err = r.Get(ctx, req.NamespacedName, rawTest); err != nil {
		log.Error(err, "failed to get test", "name", req.NamespacedName)
		// do not requeue, the test may have been deleted or the cache is invalid
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if rawTest.Status.State.IsTerminated() {
		return ctrl.Result{}, nil
	}

	// TODO(codeblooded): Consider moving this to a mutating webhook
	test := rawTest.DeepCopy()
	if err = r.Defaults.SetLoadTestDefaults(test); err != nil {
		log.Error(err, "failed to clone test with defaults")
		test.Status.State = grpcv1.Errored
		test.Status.Reason = grpcv1.FailedSettingDefaultsError
		test.Status.Message = fmt.Sprintf("failed to reconcile tests with defaults: %v", err)

		if err = r.Status().Update(ctx, test); err != nil {
			log.Error(err, "failed to update test status when setting defaults failed")
		}

		return ctrl.Result{}, err
	}
	if !reflect.DeepEqual(rawTest, test) {
		if err = r.Update(ctx, test); err != nil {
			log.Error(err, "failed to update test with defaults")
			return ctrl.Result{}, err
		}
	}

	cfgMap := new(corev1.ConfigMap)
	if err = r.Get(ctx, req.NamespacedName, cfgMap); err != nil {
		log.Info("failed to find existing scenarios ConfigMap")

		if client.IgnoreNotFound(err) != nil {
			// The ConfigMap existence was not at issue, so this is likely an
			// issue with the Kubernetes API. So, we'll update the status, retry
			// with exponential backoff and allow the timeout to catch it.

			test.Status.State = grpcv1.Unknown
			test.Status.Reason = grpcv1.KubernetesError
			test.Status.Message = fmt.Sprintf("kubernetes error (retrying): failed to get scenarios ConfigMap: %v", err)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				log.Error(updateErr, "failed to update status after failure to get scenarios ConfigMap: %v", err)
			}

			return ctrl.Result{}, err
		}

		cfgMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
			Data: map[string]string{
				"scenarios.json": test.Spec.ScenariosJSON,
			},

			// TODO: Enable ConfigMap immutability when it becomes available
			// Immutable: optional.BoolPtr(true),
		}

		if refError := ctrl.SetControllerReference(test, cfgMap, r.Scheme); refError != nil {
			// We should retry when we cannot set a controller reference on the
			// ConfigMap. This breaks garbage collection. If left to continue
			// for manual cleanup, it could create hidden errors when a load
			// test with the same name is created.
			log.Error(refError, "could not set controller reference on scenarios ConfigMap")

			test.Status.State = grpcv1.Unknown
			test.Status.Reason = grpcv1.KubernetesError
			test.Status.Message = fmt.Sprintf("kubernetes error (retrying): could not setup garbage collection for scenarios ConfigMap: %v", refError)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				log.Error(updateErr, "failed to update status after failure to get and create scenarios ConfigMap")
			}

			return ctrl.Result{Requeue: true}, refError
		}

		if createErr := r.Create(ctx, cfgMap); createErr != nil {
			log.Error(err, "failed to create scenarios ConfigMap")
			return ctrl.Result{Requeue: true}, createErr
		}
	}

	pods := new(corev1.PodList)
	if err = r.List(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "failed to list pods", "namespace", req.Namespace)
		return ctrl.Result{Requeue: true}, err
	}

	ownedPods := status.PodsForLoadTest(test, pods.Items)
	test.Status = status.ForLoadTest(test, ownedPods)
	if err = r.Status().Update(ctx, test); err != nil {
		log.Error(err, "failed to update test status")
		return ctrl.Result{Requeue: true}, err
	}

	missingPods := status.CheckMissingPods(test, ownedPods)
	builder := podbuilder.New(r.Defaults, test)

	makePod := func(pod *corev1.Pod) (*ctrl.Result, error) {
		if err = ctrl.SetControllerReference(test, pod, r.Scheme); err != nil {
			log.Error(err, "could not set controller reference on pod, pod will not be garbage collected", "pod", pod)
			return &ctrl.Result{}, err
		}

		if err = r.Create(ctx, pod); err != nil {
			log.Error(err, "could not create new pod", "pod", pod)
			return &ctrl.Result{Requeue: true}, err
		}

		return nil, nil
	}

	for i := range missingPods.Servers {
		result, err := makePod(builder.PodForServer(&missingPods.Servers[i]))
		if result != nil {
			log.Error(err, "failed to initialize server")
			return *result, err
		}
	}
	for i := range missingPods.Clients {
		result, err := makePod(builder.PodForClient(&missingPods.Clients[i]))
		if result != nil {
			log.Error(err, "failed to initalize client")
			return *result, err
		}
	}
	if missingPods.Driver != nil {
		result, err := makePod(builder.PodForDriver(missingPods.Driver))
		if result != nil {
			log.Error(err, "failed to initialize driver")
			return *result, err
		}
	}

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
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
