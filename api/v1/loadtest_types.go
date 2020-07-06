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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: AFTER EDITS, YOU MUST RUN `make` TO REGENERATE CODE.

type Clone struct {
	// Image is the name of the container image that can clone code,
	// placing it in a /src/workspace directory.
	//
	// This field is optional. When omitted, a container that can clone
	// public GitHub repos over HTTPs is used.
	// +optional
	Image *string `json:"image,omitempty"`

	// Repo is the URL to clone a git repository. With GitHub, this should
	// end in a `.git` extension.
	// +optional
	Repo *string `json:"repo,omitempty"`

	// GitRef is a branch, tag or commit hash to checkout after a successful
	// clone. This snapshot will be the state of the code in /src/workspace.
	// +optional
	GitRef *string `json:"gitRef,omitempty"`
}

type Build struct {
	// Image is the name of the container image that can build code
	// in a /src/workspace directory.
	//
	// This field is optional when a Language is specified on its
	// parent Component. For example, a developer may specify a
	// "java" server. This field will be implicitly set to the most
	// recent supported gradle image.
	// +optional
	Image string `json:"image,omitempty"`

	// Command is the path to the executable that will build the
	// code in the /src/workspace directory. If unspecified, the
	// entrypoint for the build container is used.
	// +optional
	Command []string `json:"command,omitempty"`

	// Args provide command line arguments to the command. If a
	// command is not specified, these arguments will be ignored
	// in favor of the default arguments for container's
	// entrypoint.
	// +optional
	Args []string `json:"args,omitempty"`

	// Env are environment variables that should be set within the
	// build container. This is provided for compilers that alter
	// behavior due to certain environment variables.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

type Run struct {
	// Image is the name of the container image that provides the
	// runtime for the test component. It provides the necessary
	// interpreters, system libraries and environments to run the
	// built executable.
	//
	// This field is optional when a Language is specified on its
	// parent Component. For example, a developer may specify a
	// "python3" client. This field will be implicitly set to the
	// most recent supported python3 image.
	// +optional
	Image *string `json:"image,omitempty"`

	// Command is the path to the executable that will run the
	// component of the test. When unset, the entrypoint of the
	// container image will be used.
	// +optional
	Command []string `json:"command,omitempty"`

	// Args provide command line arguments to the command. If a
	// command is not specified, these arguments will be ignored
	// in favor of the default arguments for container's
	// entrypoint.
	// +optional
	Args []string `json:"args,omitempty"`

	// Env are environment variables that should be set within the
	// running container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// VolumeMounts permit sharing directories across containers.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

type Component struct {
	// Language is the code that identifies the programming language used by
	// the component. For example, "cxx" may represent C++.
	//
	// Specifying a language is required. Aside from metadata, It allows the
	// image field on the Build and Run objects to be inferred.
	Language string `json:"language"`

	// Clone specifies the repository and snapshot where the code can be
	// found. This is used to build and run tests.
	// +optional
	Clone *Clone `json:"clone,omitempty"`

	// Build describes how the cloned code should be built, including any
	// compiler arguments or flags.
	// +optional
	Build *Build `json:"build,omitempty"`

	// Run describes the runtime of the container during the test
	// itself. This is required, because the system must run some
	// container.
	Run Run `json:"run"`
}

type Driver struct {
	Component `json:",inline"`
}

type Server struct {
	Component `json:",inline"`
}
type Client struct {
	Component `json:",inline"`
}

type Results struct {
	// BigQueryTable names a dataset where the results of the test should
	// be stored. If omitted, no results are saved to BigQuery.
	// +optional
	BigQueryTable *string `json:"bigQueryTable,omitempty"`
}

type Scenario struct {
	// ConfigMapRef points to a Kubernetes ConfigMap which
	// specifies the parameters for the test.
	// +optional
	ConfigMapRef *corev1.ConfigMapEnvSource `json:"configMapRef,omitempty"`
}

// LoadTestSpec defines the desired state of LoadTest
type LoadTestSpec struct {
	// Driver is the component that orchestrates the test. It may be
	// unspecified, allowing the system to choose the appropriate driver.
	// +optional
	Driver *Driver `json:"driver,omitempty"`

	// Servers are a list of components that receive traffic from clients.
	// +optional
	Servers []Server `json:"servers,omitempty"`

	// Clients are a list of components that send traffic to servers.
	// +optional
	Clients []Client `json:"clients,omitempty"`

	// Results configures where the results of the test should be stored.
	// When omitted, the results will only be stored in Kubernetes for a
	// limited time.
	// +optional
	Results *Results `json:"results,omitempty"`

	// Scenarios provides a list of configurations for testing.
	// +optional
	Scenarios []Scenario `json:"scenarios,omitempty"`
}

type LoadTestState string

const (
	// UnrecognizedState indicates that the controller has not yet
	// acknowledged or started reconiling the load test.
	UnrecognizedState LoadTestState = "Unrecognized"

	// WaitingState indicates that the load test is waiting for
	// sufficient machine availability in order to be scheduled.
	WaitingState = "Waiting"

	// ProvisioningState indicates that the load test's resources
	// are being created.
	ProvisioningState = "Provisioning"

	// PendingState indicates that the load test's resources are
	// healthy. The load test will remain in this state until the
	// status of one of its resources changes.
	PendingState = "Pending"

	// FailState indicates that a resource in the load test has
	// terminated unsuccessfully.
	FailState = "Failed"

	// SuccessState indicates that a resource terminated with a
	// successful status.
	SuccessState = "Succeeded"

	// ErrorState indicates that something went wrong, preventing
	// the controller for reconciling the load test.
	ErrorState = "Error"
)

// LoadTestStatus defines the observed state of LoadTest
type LoadTestStatus struct {
	// State identifies the current state of the load test. It is
	// important to note that this state is level-based. This means
	// its transition is non-deterministic.
	State LoadTestState `json:"state"`

	// AcknowledgeTime marks when the controller first responded to
	// the load test.
	// +optional
	AcknowledgeTime *metav1.Time `json:"acknowledgeTime,omitempty"`

	// ProvisionTime marks the time when the controller began to
	// provision the resources for the load test.
	// +optional
	ProvisionTime *metav1.Time `json:"provisionTime,omitempty"`

	// PendingTime marks the time when the load test's resources
	// were found to be in the pending state.
	// +optional
	PendingTime *metav1.Time `json:"pendingTime,omitempty"`

	// TerminateTime marks the time when a resource for the load
	// test was marked as terminated.
	// +optional
	TerminateTime *metav1.Time `json:"terminateTime,omitempty"`
}

// +kubebuilder:object:root=true

// LoadTest is the Schema for the loadtests API
type LoadTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadTestSpec   `json:"spec,omitempty"`
	Status LoadTestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LoadTestList contains a list of LoadTest
type LoadTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoadTest{}, &LoadTestList{})
}
