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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

	testTTL := time.Duration(rawTest.Spec.TTLSeconds) * time.Second
	testTimeout := time.Duration(rawTest.Spec.TimeoutSeconds) * time.Second

	if testTimeout > testTTL {
		log.Info("testTTL is less than testTimeout", "testTimeout", testTimeout, "testTTL", testTTL)
	}

	if rawTest.Status.State.IsTerminated() {
		if time.Now().Sub(rawTest.Status.StartTime.Time) >= testTTL {
			log.Info("test expired, deleting", "startTime", rawTest.Status.StartTime, "testTTL", testTTL)
			if err = r.Delete(ctx, rawTest); err != nil {
				log.Error(err, "fail to delete test")
				return ctrl.Result{Requeue: true}, err
			}
		}
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

	previousStatus := test.Status
	test.Status = status.ForLoadTest(test, ownedPods)

	if err = r.Status().Update(ctx, test); err != nil {
		log.Error(err, "failed to update test status")
		return ctrl.Result{Requeue: true}, err
	}

	missingPods := status.CheckMissingPods(test, ownedPods)
	builder := podbuilder.New(r.Defaults, test)

	createPod := func(pod *corev1.Pod) (*ctrl.Result, error) {
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
		logWithServer := log.WithValues("server", missingPods.Servers[i])

		pod, err := builder.PodForServer(&missingPods.Servers[i])
		if err != nil {
			logWithServer.Error(err, "failed to construct a pod struct for supplied server struct")

			test.Status.State = grpcv1.Errored
			test.Status.Reason = grpcv1.ConfigurationError
			test.Status.Message = fmt.Sprintf("failed to construct a pod for server at index %d: %v", i, err)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logWithServer.Error(updateErr, "failed to update status after failure to construct a pod for server")
			}

			return ctrl.Result{}, err
		}

		result, err := createPod(pod)
		if result != nil && !kerrors.IsAlreadyExists(err) {
			logWithServer.Error(err, "failed to create pod for server")

			test.Status.State = grpcv1.Errored
			test.Status.Reason = grpcv1.KubernetesError
			test.Status.Message = fmt.Sprintf("failed to create pod for server at index %d: %v", i, err)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logWithServer.Error(updateErr, "failed to update status after failure to create pod for server")
			}

			return *result, err
		}
	}
	for i := range missingPods.Clients {
		logWithClient := log.WithValues("client", missingPods.Clients[i])

		pod, err := builder.PodForClient(&missingPods.Clients[i])
		if err != nil {
			logWithClient.Error(err, "failed to construct a pod struct for supplied client struct")

			test.Status.State = grpcv1.Errored
			test.Status.Reason = grpcv1.ConfigurationError
			test.Status.Message = fmt.Sprintf("failed to construct a pod for client at index %d: %v", i, err)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logWithClient.Error(updateErr, "failed to update status after failure to construct a pod for client")
			}

			return ctrl.Result{}, err
		}

		result, err := createPod(pod)
		if result != nil && !kerrors.IsAlreadyExists(err) {
			logWithClient.Error(err, "failed to create pod for client")

			test.Status.State = grpcv1.Errored
			test.Status.Reason = grpcv1.KubernetesError
			test.Status.Message = fmt.Sprintf("failed to create pod for client at index %d: %v", i, err)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logWithClient.Error(updateErr, "failed to update status after failure to create pod for client")
			}

			return *result, err
		}
	}
	if missingPods.Driver != nil {
		logWithDriver := log.WithValues("driver", missingPods.Driver)

		pod, err := builder.PodForDriver(missingPods.Driver)
		if err != nil {
			logWithDriver.Error(err, "failed to construct a pod struct for supplied driver struct")

			test.Status.State = grpcv1.Errored
			test.Status.Reason = grpcv1.ConfigurationError
			test.Status.Message = fmt.Sprintf("failed to construct a pod for driver: %v", err)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logWithDriver.Error(updateErr, "failed to update status after failure to construct a pod for driver")
			}

			return ctrl.Result{}, err
		}

		result, err := createPod(pod)
		if result != nil && !kerrors.IsAlreadyExists(err) {
			logWithDriver.Error(err, "failed to create pod for driver")

			test.Status.State = grpcv1.Errored
			test.Status.Reason = grpcv1.KubernetesError
			test.Status.Message = fmt.Sprintf("failed to create pod for driver: %v", err)

			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logWithDriver.Error(updateErr, "failed to update status after failure to create pod for driver")
			}

			return *result, err
		}
	}

	requeueTime := getRequeueTime(test, previousStatus, log)
	if requeueTime != 0 {
		return ctrl.Result{RequeueAfter: requeueTime}, nil
	}
	return ctrl.Result{}, nil
}

// getRequeueTime takes a LoadTest and its previous status, compares the
// previous status of the load test with its updated status, and returns a
// calculated requeue time. If the test has just been assigned a start time
// (i.e., it has just started), the requeue time is set to the timeout value
// specified in the LoadTest. If the test has just been assigned a stop time
// (i.e., it has just terminated), the requeue time is set to the time-to-live
// specified in the LoadTest, minus its actual running time. In other cases,
// the requeue time is set to zero.
func getRequeueTime(updatedLoadTest *grpcv1.LoadTest, previousStatus grpcv1.LoadTestStatus, log logr.Logger) time.Duration {
	requeueTime := time.Duration(0)

	if previousStatus.StartTime == nil && updatedLoadTest.Status.StartTime != nil {
		requeueTime = time.Duration(updatedLoadTest.Spec.TimeoutSeconds) * time.Second
		log.Info("just started, should be marked as error if still running at :" + time.Now().Add(requeueTime).String())
		return requeueTime
	}

	if previousStatus.StopTime == nil && updatedLoadTest.Status.StopTime != nil {
		requeueTime = time.Duration(updatedLoadTest.Spec.TTLSeconds)*time.Second - updatedLoadTest.Status.StopTime.Sub(updatedLoadTest.Status.StartTime.Time)
		log.Info("just end, should be deleted at :" + time.Now().Add(requeueTime).String())
		return requeueTime
	}

	return requeueTime
}

// SetupWithManager configures a controller-runtime manager.
func (r *LoadTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
