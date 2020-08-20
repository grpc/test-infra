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

package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/grpc/test-infra/internal/kubehelpers"
)

// TimeoutEnv is the name of the environment variable that will contain the
// maximum amount of time to wait for pods to become ready.
const TimeoutEnv = "READY_TIMEOUT"

// DefaultTimeout specifies the amount of time to wait for ready pods if the
// environment variable specified by the TimeoutEnv constant is not set.
const DefaultTimeout = 5 * time.Minute

// OutputFileEnv is the optional name of the file where the executable should
// write a comma-separated list of IP addresses. If this environment variable is
// unset, DefaultOutputFile will be used as the default.
const OutputFileEnv = "READY_OUTPUT_FILE"

// DefaultOutputFile is the name of the default file where the executable should
// write the comma-separated list of IP addresses.
const DefaultOutputFile = "/tmp/loadtest_workers"

// KubeConfigEnv is the name of the environment variable that may contain a
// path to a kubeconfig file. This environment variable does not need to be set
// when the container runs on a node in a Kubernetes cluster.
const KubeConfigEnv = "KUBE_CONFIG"

// pollInterval specifies the amount of time between subsequent requests to the
// Kubernetes API for a list of pods.
const pollInterval = 3 * time.Second

// PodLister lists pods known to a Kubernetes cluster.
type PodLister interface {
	List(metav1.ListOptions) (*corev1.PodList, error)
}

// parseSelectors accepts a slice of strings and converts them into Kubernetes
// selectors. It returns an error if any of the strings is not a valid selector.
func parseSelectors(args []string) ([]labels.Selector, error) {
	var selectors []labels.Selector

	for i, arg := range args {
		selector, err := labels.Parse(arg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse selector #%d (%s): %v", i+1, arg, err)
		}

		selectors = append(selectors, selector)
	}

	return selectors, nil
}

// isPodReady returns true if the pod has been assigned an IP address and all of
// its containers are ready.
func isPodReady(pod *corev1.Pod) bool {
	spec := &pod.Spec
	status := &pod.Status

	if status.PodIP == "" {
		return false
	}

	if len(spec.Containers) != len(status.ContainerStatuses) {
		return false
	}

	for _, cstat := range status.ContainerStatuses {
		if !cstat.Ready {
			return false
		}
	}

	return true
}

// WaitForReadyPods blocks until pods with matching label selectors are ready.
// It accepts a context, allowing a timeout or deadline to be specified. When
// all pods are ready, it returns a slice of strings with their IP addresses.
// The order will match the label selectors.
//
// The syntax for the selectors is defined in the Parse function documented at
// pkg.go.dev/k8s.io/apimachinery/pkg/labels.
//
// If the timeout is exceeded or there is a problem communicating with the
// Kubernetes API, an error is returned.
func WaitForReadyPods(ctx context.Context, pl PodLister, args []string) ([]string, error) {
	timeoutsEnabled := true
	deadline, ok := ctx.Deadline()
	if !ok {
		timeoutsEnabled = false
		log.Printf("no timeout is set; this could block forever")
	}

	selectors, err := parseSelectors(args)
	if err != nil {
		return nil, err
	}

	var podIPs []string
	for range selectors {
		podIPs = append(podIPs, "")
	}

	matchCount := 0
	matchingPods := make(map[string]bool)

	for {
		if timeoutsEnabled && time.Now().After(deadline) {
			return nil, errors.New("timeout exceeded")
		}

		podList, err := pl.List(metav1.ListOptions{})
		if err != nil {
			log.Fatalf("failed to fetch list of pods: %v", err)
		}

		for _, pod := range podList.Items {
			if !isPodReady(&pod) {
				continue
			}

			if _, alreadyMatched := matchingPods[pod.Name]; alreadyMatched {
				continue
			}

			for i, selector := range selectors {
				if podIPs[i] != "" {
					continue
				}

				if selector.Matches(labels.Set(pod.Labels)) {
					podIPs[i] = pod.Status.PodIP
					matchingPods[pod.Name] = true
					matchCount++
					break
				}
			}
		}

		if matchCount == len(selectors) {
			break
		}

		time.Sleep(pollInterval)
	}

	return podIPs, nil
}

func main() {
	var err error
	timeout := DefaultTimeout
	timeoutStr, ok := os.LookupEnv(TimeoutEnv)
	if ok {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			log.Fatalf("failed to parse $%s: %v", TimeoutEnv, err)
		}
	}

	var clientset kubernetes.Interface
	kubeConfigFile, ok := os.LookupEnv(KubeConfigEnv)
	if ok {
		clientset, err = kubehelpers.ConnectWithConfig(kubeConfigFile)
		if err != nil {
			log.Fatalf("failed to read kubeconfig: %v", err)
		}
	} else {
		clientset, err = kubehelpers.ConnectWithinCluster()
		if err != nil {
			log.Fatalf("failed to connect with implicit kubeconfig: %v", err)
		}
	}

	outputFile := DefaultOutputFile
	outputFileOverride, ok := os.LookupEnv(OutputFileEnv)
	if ok {
		outputFile = outputFileOverride
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	podIPs, err := WaitForReadyPods(ctx, clientset.CoreV1().Pods(metav1.NamespaceAll), os.Args[1:])
	if err != nil {
		log.Fatalf("failed to wait for ready pods: %v", err)
	}

	log.Printf("all pods ready, exiting successfully")
	workerFileBody := strings.Join(podIPs, ",")
	ioutil.WriteFile(outputFile, []byte(workerFileBody), 0777)
}
