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

// NOTE: AFTER EDITS, YOU MUST RUN `make manifests` AND `make` TO REGENERATE
// CODE.

// Clone defines expectations regarding which repository and snapshot the test
// should use.
type Clone struct {
	// Image is the name of the container image that can clone code, placing
	// it in a /src/workspace directory.
	//
	// This field is optional. When omitted, a container that can clone
	// public GitHub repos over HTTPs is used.
	// +optional
	Image *string `json:"image,omitempty"`

	// Repo is the URL to clone a git repository. With GitHub, this should
	// end in a `.git` extension.
	// +optional
	Repo *string `json:"repo,omitempty"`

	// GitRef is a branch, tag or commit hash to checkout after a
	// successful clone. This will be the version of the code in the
	// /src/workspace directory.
	// +optional
	GitRef *string `json:"gitRef,omitempty"`
}

// Build defines expectations regarding which container image,
// command, arguments and environment variables are used to build the
// component.
type Build struct {
	// Image is the name of the container image that can build code,
	// placing an executable in the /src/workspace directory.
	//
	// This field is optional when a Language is specified on the
	// Component. For example, a developer may specify a "java" server.
	// Then, this image will default to the most recent gradle image.
	// +optional
	Image *string `json:"image,omitempty"`

	// Command is the path to the executable that will build the code in
	// the /src/workspace directory. If unspecified, the entrypoint for
	// the container is used.
	// +optional
	Command []string `json:"command,omitempty"`

	// Args provide command line arguments to the command. If a command
	// is not specified, these arguments will be ignored in favor of the
	// default arguments for container's entrypoint.
	// +optional
	Args []string `json:"args,omitempty"`

	// Env are environment variables that should be set within the build
	// container. This is provided for compilers that alter behavior due
	// to certain environment variables.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// Driver defines a component that orchestrates the server and clients in the
// test.
type Driver struct {
	// Name is a string that uniquely names this driver. Since load tests only
	// support one driver, it is not recommended to set this field. If no name
	// is explicitly provided, the operator will assign one.
	// +optional
	Name *string `json:"name,omitempty"`

	// Language is the code that identifies the programming language used by the
	// driver. For example, "cxx" may represent C++.
	//
	// Specifying a language is required. If the language is unknown to the
	// operator, a user must manually set a run image. If the user intends for
	// the operator to clone and build code, it must also manually set a build
	// image.
	Language string `json:"language"`

	// Pool specifies the name of the set of nodes where this driver should be
	// scheduled. If unset, the controller will choose a pool based on defaults.
	// +optional
	Pool *string `json:"pool,omitempty"`

	// Clone specifies the repository and snapshot where the code for the driver
	// can be found. This is used to test alternative implementations for the
	// driver. Most often, this will not be set. When unset, the operator will
	// use a default driver that is prebuilt.
	// +optional
	Clone *Clone `json:"clone,omitempty"`

	// Build describes how the cloned code should be built, including any
	// compiler arguments or flags. This field is only necessary if the output
	// from the clone container must be pre-processed before running the tests
	// in the run container.
	//
	// When build is specified on a test, the operator will use the driver's
	// language to find a container with a compiler for that language. If the
	// language is unknown to the operator, a user must include a custom docker
	// image.
	//
	// Note that it does not usually make sense to include build instructions
	// without clone instructions. If doing so, the build container must include
	// its input and write its output into the /src/workspace directory for the
	// run container to access it.
	// +optional
	Build *Build `json:"build,omitempty"`

	// Run describes a list of run containers. The container for the test driver is always
	// the first container on the list.
	Run []corev1.Container `json:"run"`
}

// Server defines a component that receives traffic from a set of client
// components.
type Server struct {
	// Name is a string that distinguishes this server from others in the test.
	// Since tests are currently limited to one server, setting this field is not
	// recommended. set this field. If no name is explicitly provided, the
	// operator will assign one.
	// +optional
	Name *string `json:"name,omitempty"`

	// Language is the code that identifies the programming language used by the
	// server. For example, "java" may represent Java.
	//
	// Specifying a language is required. If the language is unknown to the
	// operator, a user must manually set a run image. If the user intends for
	// the operator to clone and build code, it must also manually set a build
	// image.
	Language string `json:"language"`

	// Pool specifies the name of the set of nodes where this server should be
	// scheduled. If unset, the controller will choose a pool based on defaults.
	// +optional
	Pool *string `json:"pool,omitempty"`

	// Clone specifies the repository and snapshot where the code for the server
	// can be found. This field should not be set if the code has been prebuilt
	// in the run image.
	// +optional
	Clone *Clone `json:"clone,omitempty"`

	// Build describes how the cloned code should be built, including any
	// compiler arguments or flags. This field is only necessary if the output
	// from the clone container must be pre-processed before running the tests
	// in the run container.
	//
	// When build is specified on a test, the operator will use the server's
	// language to find a container with a compiler for that language. If the
	// language is unknown to the operator, a user must include a custom docker
	// image.
	//
	// Note that it does not usually make sense to include build instructions
	// without clone instructions. If doing so, the build container must include
	// its input and write its output into the /src/workspace directory for the
	// run container to access it.
	// +optional
	Build *Build `json:"build,omitempty"`

	// Run describes a list of run containers. The container for the test server is always
	// the first container on the list.
	Run []corev1.Container `json:"run"`

	MetricsPort int32 `json:"metricsPort,omitempty"`
}

// Client defines a component that sends traffic to a server component.
type Client struct {
	// Name is a string that distinguishes this client from others in the test.
	// Explicitly setting a name is recommended when it is helpful to
	// differentiate between multiple clients. For example, a test may use
	// clients with different settings.
	//
	// Most often, this field will not be set. When unset, the operator will
	// assign a name to the client.
	// +optional
	Name *string `json:"name,omitempty"`

	// Language is the code that identifies the programming language used by the
	// client. For example, "go" may represent Go.
	//
	// Specifying a language is required. If the language is unknown to the
	// operator, a user must manually set a run image. If the user intends for
	// the operator to clone and build code, it must also manually set a build
	// image.
	Language string `json:"language"`

	// Pool specifies the name of the set of nodes where this client should be
	// scheduled. If unset, the controller will choose a pool based on defaults.
	// +optional
	Pool *string `json:"pool,omitempty"`

	// Clone specifies the repository and snapshot where the code for the client
	// can be found. This field should not be set if the code has been prebuilt
	// in the run image.
	// +optional
	Clone *Clone `json:"clone,omitempty"`

	// Build describes how the cloned code should be built, including any
	// compiler arguments or flags. This field is only necessary if the output
	// from the clone container must be pre-processed before running the tests
	// in the run container.
	//
	// When build is specified on a test, the operator will use the client's
	// language to find a container with a compiler for that language. If the
	// language is unknown to the operator, a user must include a custom docker
	// image.
	//
	// Note that it does not usually make sense to include build instructions
	// without clone instructions. If doing so, the build container must include
	// its input and write its output into the /src/workspace directory for the
	// run container to access it.
	// +optional
	Build *Build `json:"build,omitempty"`

	// Run describes a list of run containers. The container for the test client is always
	// the first container on the list.
	Run []corev1.Container `json:"run"`

	MetricsPort int32 `json:"metricsPort,omitempty"`
}

// Results defines where and how test results and artifacts should be
// stored.
type Results struct {
	// BigQueryTable names a dataset where the results of the test
	// should be stored. If omitted, no results are saved to BigQuery.
	// +optional
	BigQueryTable *string `json:"bigQueryTable,omitempty"`
}

// LoadTestSpec defines the desired state of LoadTest
type LoadTestSpec struct {
	// Driver is the component that orchestrates the test. It may be
	// unspecified, allowing the system to choose the appropriate driver.
	// +optional
	Driver *Driver `json:"driver,omitempty"`

	// Servers are a list of components that receive traffic from
	// clients.
	// +optional
	Servers []Server `json:"servers,omitempty"`

	// Clients are a list of components that send traffic to servers.
	// +optional
	Clients []Client `json:"clients,omitempty"`

	// Results configures where the results of the test should be
	// stored. When omitted, the results will only be stored in
	// Kubernetes for a limited time.
	// +optional
	Results *Results `json:"results,omitempty"`

	// ScenariosJSON is string with the contents of a Scenarios message,
	// formatted as JSON. See the Scenarios protobuf definition for details:
	// https://github.com/grpc/grpc-proto/blob/master/grpc/testing/control.proto.
	// +optional
	ScenariosJSON string `json:"scenariosJSON,omitempty"`

	// Timeout provides the longest running time allowed for a LoadTest.
	// +kubebuilder:validation:Minimum:=1
	TimeoutSeconds int32 `json:"timeoutSeconds"`

	// TTL provides the longest time a LoadTest can live on the cluster.
	// +kubebuilder:validation:Minimum:=1
	TTLSeconds int32 `json:"ttlSeconds"`
}

// LoadTestState reflects the derived state of the load test from its
// components. If any one component has errored, the load test will be marked in
// an Errored state, too. This will occur even if the other components are
// running or succeeded.
// +kubebuilder:default=Unknown
type LoadTestState string

const (
	// Unknown states indicate that the load test is in an indeterminate state.
	// Something may have gone wrong, but it may be recoverable. No assumption
	// should be made about the next state. It may transition to any other state
	// or remain Unknown until a timeout occurs.
	Unknown LoadTestState = "Unknown"

	// Initializing states indicate that load test's pods are under construction.
	// This may mean that code is being cloned, built or assembled.
	Initializing LoadTestState = "Initializing"

	// Running states indicate that the initialization for a load test's pods has
	// completed successfully. The run container has started.
	Running LoadTestState = "Running"

	// Succeeded states indicate the driver pod's run container has terminated
	// successfully, signaled by a zero exit code.
	Succeeded LoadTestState = "Succeeded"

	// Errored states indicate the load test encountered a problem that prevented
	// a successful run.
	Errored LoadTestState = "Errored"
)

// IsTerminated returns true if the test has finished due to a success, failure
// or error. Otherwise, it returns false.
func (lts LoadTestState) IsTerminated() bool {
	return lts == Succeeded || lts == Errored
}

// InitContainerError is the reason string when an init container has failed on
// one of the load test's pods.
var InitContainerError = "InitContainerError"

// ContainerError is the reason string when a container has failed on one of the
// load test's pods.
var ContainerError = "ContainerError"

// FailedSettingDefaultsError is the reason string when defaults could not be
// set on a load test.
var FailedSettingDefaultsError = "FailedSettingDefaults"

// ConfigurationError is the reason string when a LoadTest spec is invalid.
var ConfigurationError = "ConfigurationError"

// PodsMissing is the reason string when the load test is missing pods and is still
// in the Initializing state.
var PodsMissing = "PodsMissing"

// PoolError is the reason string when a driver, client or server requires nodes
// from a nonexistent pool.
var PoolError = "PoolError"

// TimeoutErrored is the reason string when the load test has not yet terminated
// but exceeded the timeout.
var TimeoutErrored = "TimeoutErrored"

// KubernetesError is the reason string when an issue occurs with Kubernetes
// that is not known to be directly related to a load test.
var KubernetesError = "KubernetesError"

// LoadTestStatus defines the observed state of LoadTest
type LoadTestStatus struct {
	// State identifies the current state of the load test. It is
	// important to note that this state is level-based. This means its
	// transition is non-deterministic.
	State LoadTestState `json:"state"`

	// Reason is a camel-case string that indicates the reasoning behind the
	// current state.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human legible string that describes the current state.
	// +optional
	Message string `json:"message,omitempty"`

	// StartTime is the time when the controller first reconciled the load test.
	// It is maintained in a best-attempt effort; meaning, it is not guaranteed to
	// be correct.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// StopTime is the time when the controller last entered the Succeeded,
	// Failed or Errored states.
	// +optional
	StopTime *metav1.Time `json:"stopTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// LoadTest is the Schema for the loadtests API
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
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
