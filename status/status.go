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

// Package status contains code for determining the current state of
// the world, including the health of a load test and its resources.
package status

import (
	"fmt"
	"strings"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// State reflects the observed state of a resource.
type State string

const (
	// Pending indicates that the resource has not yet been observed as
	// succeeding or failing.
	Pending State = "Pending"

	// Succeeded indicates that the resource has terminated successfully,
	// marked by a zero exit code.
	Succeeded State = "Succeeded"

	// Errored indicates that the resource has terminated unsuccessfully,
	// marked by a non-zero exit code.
	Errored State = "Failed"
)

// StateForContainerStatus accepts the status of a container and returns a
// ContainerState and a pointer to the integer exit code. If the container has
// not terminated, a Pending state and nil pointer are returned.
func StateForContainerStatus(status *corev1.ContainerStatus) (State, *int32) {
	if terminateState := status.State.Terminated; terminateState != nil {
		var state State = Errored

		if terminateState.ExitCode == 0 {
			state = Succeeded
		}

		return state, &terminateState.ExitCode
	}

	if waitState := status.State.Waiting; waitState != nil {
		if strings.Compare("CrashLoopBackOff", waitState.Reason) == 0 {
			return Errored, nil
		}
	}

	return Pending, nil
}

// StateForPodStatus accepts the status of a pod and returns a State, as well
// as the reason and message. The reason is a camel-case word that is machine
// comparable. The message is a human-legible description. If the pod has not
// terminated or it terminated successfully, the reason and message strings will
// be empty.
func StateForPodStatus(status *corev1.PodStatus) (state State, reason string, message string) {
	podState := Pending

	for i := range status.InitContainerStatuses {
		initContStat := &status.InitContainerStatuses[i]
		contState, exitCode := StateForContainerStatus(initContStat)

		if contState == Errored {
			message := fmt.Sprintf("init container %q terminated with exit code %d", initContStat.Name, *exitCode)
			return Errored, grpcv1.InitContainerError, message
		}
	}

	for i := range status.ContainerStatuses {
		contStat := &status.ContainerStatuses[i]
		contState, exitCode := StateForContainerStatus(contStat)

		if contState == Errored {
			message := fmt.Sprintf("container %q terminated with exit code %d", contStat.Name, *exitCode)
			return Errored, grpcv1.ContainerError, message
		}

		if contState != Pending {
			podState = contState
		}
	}

	return podState, "", ""
}
