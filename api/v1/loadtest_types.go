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

// Run defines expectations regarding the runtime environment for the
// test component itself.
type Run struct {
	// Image is the name of the container image that provides the
	// runtime for the test component.
	//
	// This field is optional when a Language is specified on the
	// Component. For example, a developer may specify a "python3"
	// client. This field will be implicitly set to the most recent
	// supported python3 image.
	// +optional
	Image *string `json:"image,omitempty"`

	// Command is the path to the executable that will run the component
	// of the test. When unset, the entrypoint of the container image
	// will be used.
	// +optional
	Command []string `json:"command,omitempty"`

	// Args provide command line arguments to the command.
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

// Component defines a runnable unit of the test.
type Component struct {
	// Name is a string which uniquely identifies the component when
	// compared to other components in the load test. If omitted, the
	// system will assign a globally unique name.
	// +optional
	Name *string `json:"name,omitempty"`

	// Language is the code that identifies the programming language used by
	// the component. For example, "cxx" may represent C++.
	//
	// Specifying a language is required. Aside from metadata, It allows the
	// image field on the Build and Run objects to be inferred.
	Language string `json:"language"`

	// Pool specifies the name of the set of nodes where this component should
	// be scheduled. If unset, the controller will choose a pool based on the
	// type of component.
	Pool *string `json:"pool,omitempty"`

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

// Driver defines a component that orchestrates the server and clients
// in the test.
type Driver struct {
	Component `json:",inline"`
}

// Server defines a component that receives traffic from a set of client
// components.
type Server struct {
	Component `json:",inline"`
}

// Client defines a component that sends traffic to a server component.
type Client struct {
	Component `json:",inline"`
}

// Results defines where and how test results and artifacts should be
// stored.
type Results struct {
	// BigQueryTable names a dataset where the results of the test
	// should be stored. If omitted, no results are saved to BigQuery.
	// +optional
	BigQueryTable *string `json:"bigQueryTable,omitempty"`
}

// Scenario references a ConfigMap with the configuration for the driver
// and the server clients under test.
type Scenario struct {
	// Name identifies the name of the ConfigMap with the scenario data.
	Name string `json:"name"`
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

	// Scenarios provides a list of configurations for testing.
	// +optional
	Scenarios []Scenario `json:"scenarios,omitempty"`
}

// LoadTestState reflects the derived state of the load test from its
// components. If any one component has errored or failed, the load test will
// be marked in a Failed or Errored state, too. This will occur even if the
// other components are running or succeeded.
// +kubebuilder:default=Unrecognized
type LoadTestState string

const (
	// Initializing states indicate that load test's pods are under construction.
	// This may mean that code is being cloned, built or assembled.
	Initializing LoadTestState = "Initializing"

	// Running states indicate that the initialization for a load test's pods has
	// completed successfully. The run container has started.
	Running = "Running"

	// Succeeded states indicate the driver pod's run container has terminated
	// successfully, signaled by a zero exit code.
	Succeeded = "Succeeded"

	// Failed states indicate the driver pod's run container has terminated
	// unsuccessfully, signaled by a non-zero exit code.
	//
	// The Failed state is different from the Errored state, because Failed
	// guarantees that the problem originated in the driver at runtime. An Errored
	// state indicates any other problem, such as an inability to compile the
	// driver or a failed worker..
	Failed = "Failed"

	// Errored states indicate the load test failed to run successfully. This may
	// signal a problem with the initialization of a load test, including
	// cloning and compiling, or the runtime of one of the workers.
	Errored = "Errored"
)

// InitContainerError is the reason string when an init container has failed on
// one of the load test's pods.
var InitContainerError = "InitContainerError"

// ContainerError is the reason string when a container has failed on one of the
// load test's pods.
var ContainerError = "ContainerError"

// PodsMissing is the reason string when the load test is missing pods and is still
// in the Initializing state.
var PodsMissing = "PodsMissing"

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
