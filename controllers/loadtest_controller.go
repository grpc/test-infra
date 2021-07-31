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
	"errors"
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
	"sigs.k8s.io/controller-runtime/pkg/log"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/podbuilder"
	"github.com/grpc/test-infra/status"
)

var (
	errCacheSync       = errors.New("failed to sync cache")
	errNonexistentPool = errors.New("pool does not exist")
)

// LoadTestReconciler reconciles a LoadTest object
type LoadTestReconciler struct {
	client.Client
	mgr      ctrl.Manager
	Defaults *config.Defaults
	Scheme   *runtime.Scheme
}

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=nodes/status,verbs=get
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

// Reconcile attempts to bring the current state of the load test into agreement
// with its declared spec. This may mean provisioning resources, doing nothing
// or handling the termination of its pods.
func (r *LoadTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	logger := log.FromContext(ctx).WithValues("loadtest", req.NamespacedName)

	rawTest := new(grpcv1.LoadTest)
	if err = r.Get(ctx, req.NamespacedName, rawTest); err != nil {
		logger.Error(err, "failed to get test", "name", req.NamespacedName)
		err = client.IgnoreNotFound(err)
		return ctrl.Result{Requeue: err != nil}, err
	}

	testTTL := time.Duration(rawTest.Spec.TTLSeconds) * time.Second
	testTimeout := time.Duration(rawTest.Spec.TimeoutSeconds) * time.Second

	if testTimeout > testTTL {
		logger.Info("testTTL is less than testTimeout", "testTimeout", testTimeout, "testTTL", testTTL)
	}

	if rawTest.Status.State.IsTerminated() {
		if time.Now().Sub(rawTest.Status.StartTime.Time) >= testTTL {
			logger.Info("test expired, deleting", "startTime", rawTest.Status.StartTime, "testTTL", testTTL)
			if err = r.Delete(ctx, rawTest); err != nil {
				logger.Error(err, "fail to delete test")
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{Requeue: false}, nil
	}

	// TODO(codeblooded): Consider moving this to a mutating webhook
	test := rawTest.DeepCopy()
	if err = r.Defaults.SetLoadTestDefaults(test); err != nil {
		logger.Error(err, "failed to clone test with defaults")
		test.Status.State = grpcv1.Errored
		test.Status.Reason = grpcv1.FailedSettingDefaultsError
		test.Status.Message = fmt.Sprintf("failed to reconcile tests with defaults: %v", err)
		if err = r.Status().Update(ctx, test); err != nil {
			logger.Error(err, "failed to update test status when setting defaults failed")
		}
		return ctrl.Result{Requeue: false}, nil
	}
	if !reflect.DeepEqual(rawTest, test) {
		if err = r.Update(ctx, test); err != nil {
			logger.Error(err, "failed to update test with defaults")
			return ctrl.Result{Requeue: true}, err
		}
	}

	cfgMap := new(corev1.ConfigMap)
	if err = r.Get(ctx, req.NamespacedName, cfgMap); err != nil {
		logger.Info("failed to find existing scenarios ConfigMap")

		if client.IgnoreNotFound(err) != nil {
			// The ConfigMap existence was not at issue, so this is likely an
			// issue with the Kubernetes API. So, we'll update the status, retry
			// with exponential backoff and allow the timeout to catch it.
			test.Status.State = grpcv1.Unknown
			test.Status.Reason = grpcv1.KubernetesError
			test.Status.Message = fmt.Sprintf("kubernetes error (retrying): failed to get scenarios ConfigMap: %v", err)
			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logger.Error(updateErr, "failed to update status after failure to get scenarios ConfigMap: %v", err)
			}
			return ctrl.Result{Requeue: true}, err
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
			logger.Error(refError, "could not set controller reference on scenarios ConfigMap")
			test.Status.State = grpcv1.Unknown
			test.Status.Reason = grpcv1.KubernetesError
			test.Status.Message = fmt.Sprintf("kubernetes error (retrying): could not setup garbage collection for scenarios ConfigMap: %v", refError)
			if updateErr := r.Status().Update(ctx, test); updateErr != nil {
				logger.Error(updateErr, "failed to update status after failure to get and create scenarios ConfigMap")
			}
			return ctrl.Result{Requeue: true}, refError
		}

		if createErr := r.Create(ctx, cfgMap); createErr != nil {
			logger.Error(err, "failed to create scenarios ConfigMap")
			return ctrl.Result{Requeue: true}, createErr
		}
	}

	pods := new(corev1.PodList)
	if err = r.List(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
		logger.Error(err, "failed to list pods", "namespace", req.Namespace)
		return ctrl.Result{Requeue: true}, err
	}
	ownedPods := status.PodsForLoadTest(test, pods.Items)

	previousStatus := test.Status
	test.Status = status.ForLoadTest(test, ownedPods)
	if err = r.Status().Update(ctx, test); err != nil {
		// Racing conditions arises when multiple threads tried to update the status
		// of the same object. Since Kubernetes' control loop is edge-triggered and
		// level-driven, if the update frequency is high, during the time the
		// previous thread is updating the status of the LOadTest, the subsequent
		// thread can also attempt the same update, however the
		// base the later thread read before was already updated by the previous
		// thread. This situation causes a conflict error. Iince the LoadTest status
		// is already updated, this error is not a real, not requeue this
		// reconciliation would not hurt the function of our current controller.
		if kerrors.IsConflict(err) {
			logger.Info("racing condition arises when multiple threads attempt to update the status of the same LoadTest")
			return ctrl.Result{Requeue: false}, nil
		}
		logger.Error(err, "failed to update test status")
		return ctrl.Result{Requeue: true}, err
	}

	missingPods := status.CheckMissingPods(test, ownedPods)
	if !missingPods.IsEmpty() {
		if !r.mgr.GetCache().WaitForCacheSync(ctx) {
			logger.Error(errCacheSync, "could not invalidate the cache which is required to gang schedule")
			return ctrl.Result{Requeue: true}, errCacheSync
		}

		nodes := new(corev1.NodeList)
		if err = r.List(ctx, nodes); err != nil {
			logger.Error(err, "failed to list nodes")
			return ctrl.Result{Requeue: true}, err
		}

		// since we are attempting to schedule and have invalidated the cache,
		// we need to reload the pods for any missed changes
		pods = new(corev1.PodList)
		if err = r.List(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
			logger.Error(err, "failed to list pods", "namespace", req.Namespace)
			return ctrl.Result{Requeue: true}, err
		}

		// perform one final check to make sure the pods are still missing
		missingPods = status.CheckMissingPods(test, status.PodsForLoadTest(test, pods.Items))
		if missingPods.IsEmpty() {
			goto setRequeueTime
		}

		var defaultClientPool string
		var defaultDriverPool string
		var defaultServerPool string
		poolCapacities := make(map[string]int)
		for _, node := range nodes.Items {
			pool, ok := node.Labels[config.PoolLabel]
			if !ok {
				logger.Info("encountered a node without a pool label", "nodeName", node.Name)
				continue
			}

			if defaultPoolLabels := r.Defaults.DefaultPoolLabels; defaultPoolLabels != nil {
				if defaultClientPool == "" {
					if _, ok := node.Labels[defaultPoolLabels.Client]; ok {
						defaultClientPool = pool
					}
				}
				if defaultDriverPool == "" {
					if _, ok := node.Labels[defaultPoolLabels.Driver]; ok {
						defaultDriverPool = pool
					}
				}
				if defaultServerPool == "" {
					if _, ok := node.Labels[defaultPoolLabels.Server]; ok {
						defaultServerPool = pool
					}
				}

				if _, ok = poolCapacities[pool]; !ok {
					poolCapacities[pool] = 0
				}
			}

			poolCapacities[pool]++
		}

		poolAvailabilities := make(map[string]int)
		for pool, capacity := range poolCapacities {
			poolAvailabilities[pool] = capacity
		}
		for _, pod := range pods.Items {
			pool, ok := pod.Labels[config.PoolLabel]
			if !ok {
				logger.Info("encountered a pod without a pool label", "pod", pod)
				continue
			}
			if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
				poolAvailabilities[pool]--
			}
		}

		adjustAvailabilityForDefaults := func(defaultPoolKey, defaultPoolName string) bool {
			if c, ok := missingPods.NodeCountByPool[defaultPoolKey]; ok && c > 0 {
				if defaultPoolName == "" {
					logger.Error(errNonexistentPool, "default pool is not defined or does not existed in the cluster", "requestedDefaultPool", defaultPoolKey)
					test.Status.State = grpcv1.Errored
					test.Status.Reason = grpcv1.PoolError
					test.Status.Message = fmt.Sprintf("default pool %q is not defined or does not existed in the cluster", defaultPoolKey)
					if updateErr := r.Status().Update(ctx, test); updateErr != nil {
						logger.Error(updateErr, "failed to update status after failure due to requesting nodes from a nonexistent pool")
					}
					return false
				}
				missingPods.NodeCountByPool[defaultPoolName] += c
			}
			delete(missingPods.NodeCountByPool, defaultPoolKey)
			return true
		}
		if ok := adjustAvailabilityForDefaults(status.DefaultClientPool, defaultClientPool); !ok {
			return ctrl.Result{Requeue: false}, nil
		}
		if ok := adjustAvailabilityForDefaults(status.DefaultDriverPool, defaultDriverPool); !ok {
			return ctrl.Result{Requeue: false}, nil
		}
		if ok := adjustAvailabilityForDefaults(status.DefaultServerPool, defaultServerPool); !ok {
			return ctrl.Result{Requeue: false}, nil
		}

		for pool, requiredNodeCount := range missingPods.NodeCountByPool {
			availableNodeCount, ok := poolAvailabilities[pool]
			if !ok {
				logger.Error(errNonexistentPool, "requested pool does not exist and cannot be considered when scheduling", "requestedPool", pool)
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.PoolError
				test.Status.Message = fmt.Sprintf("requested pool %q does not exist", pool)
				if updateErr := r.Status().Update(ctx, test); updateErr != nil {
					logger.Error(updateErr, "failed to update status after failure due to requesting nodes from a nonexistent pool")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			if requiredNodeCount > availableNodeCount {
				logger.Info("cannot schedule test: inadequate availability for pool", "pool", pool, "requiredNodeCount", requiredNodeCount, "availableNodeCount", availableNodeCount)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}
		}

		builder := podbuilder.New(r.Defaults, test)
		createPod := func(pod *corev1.Pod) (*ctrl.Result, error) {
			if err = ctrl.SetControllerReference(test, pod, r.Scheme); err != nil {
				logger.Error(err, "could not set controller reference on pod, pod will not be garbage collected", "pod", pod)
				return &ctrl.Result{Requeue: true}, err
			}

			if err = r.Create(ctx, pod); err != nil {
				logger.Error(err, "could not create new pod", "pod", pod)
				return &ctrl.Result{Requeue: true}, err
			}

			return nil, nil
		}

		for i := range missingPods.Servers {
			logWithServer := logger.WithValues("server", missingPods.Servers[i])

			pod, err := builder.PodForServer(&missingPods.Servers[i])
			if err != nil {
				logWithServer.Error(err, "failed to construct a pod struct for supplied server struct")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.ConfigurationError
				test.Status.Message = fmt.Sprintf("failed to construct a pod for server at index %d: %v", i, err)
				if updateErr := r.Status().Update(ctx, test); updateErr != nil {
					logWithServer.Error(updateErr, "failed to update status after failure to construct a pod for server")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			if missingPods.Servers[i].Pool == nil {
				pod.Labels[config.PoolLabel] = defaultServerPool
			} else {
				pod.Labels[config.PoolLabel] = *missingPods.Servers[i].Pool
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
			logWithClient := logger.WithValues("client", missingPods.Clients[i])

			pod, err := builder.PodForClient(&missingPods.Clients[i])
			if err != nil {
				logWithClient.Error(err, "failed to construct a pod struct for supplied client struct")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.ConfigurationError
				test.Status.Message = fmt.Sprintf("failed to construct a pod for client at index %d: %v", i, err)
				if updateErr := r.Status().Update(ctx, test); updateErr != nil {
					logWithClient.Error(updateErr, "failed to update status after failure to construct a pod for client")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			if missingPods.Clients[i].Pool == nil {
				pod.Labels[config.PoolLabel] = defaultClientPool
			} else {
				pod.Labels[config.PoolLabel] = *missingPods.Clients[i].Pool
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
			logWithDriver := logger.WithValues("driver", missingPods.Driver)

			pod, err := builder.PodForDriver(missingPods.Driver)
			if err != nil {
				logWithDriver.Error(err, "failed to construct a pod struct for supplied driver struct")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.ConfigurationError
				test.Status.Message = fmt.Sprintf("failed to construct a pod for driver: %v", err)
				if updateErr := r.Status().Update(ctx, test); updateErr != nil {
					logWithDriver.Error(updateErr, "failed to update status after failure to construct a pod for driver")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			if missingPods.Driver.Pool == nil {
				pod.Labels[config.PoolLabel] = defaultDriverPool
			} else {
				pod.Labels[config.PoolLabel] = *missingPods.Driver.Pool
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
	}

setRequeueTime:
	requeueTime := getRequeueTime(test, previousStatus, logger)
	if requeueTime != 0 {
		return ctrl.Result{RequeueAfter: requeueTime}, nil
	}

	return ctrl.Result{Requeue: false}, nil
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
	r.mgr = mgr
	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
