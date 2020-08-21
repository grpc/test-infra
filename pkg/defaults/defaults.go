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

package defaults

const (
	// LoadTestLabel is a label which contains the test's unique name.
	LoadTestLabel = "loadtest"

	// RoleLabel is a label with the role  of a test component. For
	// example, "loadtest-role=server" indicates a server component.
	RoleLabel = "loadtest-role"

	// ComponentNameLabel is a label used to distinguish between test
	// components with the same role.
	ComponentNameLabel = "loadtest-component"

	// ServerRole is the value the controller expects for the RoleLabel
	// on a server component.
	ServerRole = "server"

	// ClientRole is the value the controller expects for the RoleLabel
	// on a client component.
	ClientRole = "client"

	// DriverRole is the value the controller expects for the RoleLabel
	// on a driver component.
	DriverRole = "driver"
)

// LanguageDefault defines a programming language, as well as its
// default container images.
type LanguageDefault struct {
	// Language uniquely identifies a programming language. When the
	// system encounters this name, it will select the build image and
	// run image as the defaults.
	Language string `json:"language"`

	// BuildImage specifies the default container image for building or
	// assembling an executable or bundle for a language. This image
	// likely contains a compiler and any required libraries for
	// compilation.
	BuildImage string `json:"buildImage"`

	// RunImage specifies the default container image for the
	// environment for the runtime of the test. It should provide any
	// necessary interpreters or dependencies to run or use the output
	// of the build image.
	RunImage string `json:"runImage"`
}

// Defaults defines the default settings for the system.
type Defaults struct {
	// DriverPool is the name of a pool where driver components should
	// be scheduled by default.
	DriverPool string `json:"driverPool"`

	// WorkerPool is the name of a pool where server and client
	// components should be scheduled by default.
	WorkerPool string `json:"workerPool"`

	// DriverPort is the port through which the driver and workers
	// communicate.
	DriverPort int32 `json:"driverPort"`

	// ServerPort is the port through which a server component accepts
	// traffic from a client component.
	ServerPort int32 `json:"serverPort"`

	// CloneImage specifies the default container image to use for
	// cloning Git repositories at a specific snapshot.
	CloneImage string `json:"cloneImage"`

	// DriverImage specifies a default driver image. This image will
	// be used to orchestrate a test.
	DriverImage string `json:"driverImage"`

	// Languages specifies the default build and run container images
	// for each known language.
	Languages []LanguageDefault `json:"languages,omitempty"`
}
