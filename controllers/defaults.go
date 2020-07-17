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

const (
	// LoadTestLabel is a label which contains the test's unique name.
	LoadTestLabel = "loadtest"

	// RoleLabel is a label with the role  of a test component. For
	// example, "loadtest-role=server" indicates a server component.
	RoleLabel = "loadtest-role"

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

type ControllerDefaults struct {
	// DriverPool is the name of a pool where driver components should
	// be scheduled by default.
	DriverPool string

	// WorkerPool is the name of a pool where server and client
	// components should be scheduled by default.
	WorkerPool string

	// DriverPort is the port through which the driver and workers
	// communicate.
	DriverPort int32

	// ServerPort is the port through which a server component accepts
	// traffic from a client component.
	ServerPort int32

	// CloneImage specifies the default container image to use for
	// cloning Git repositories at a specific snapshot.
	CloneImage string

	// BuildImages specifies the default container image for building
	// each language. This image should contain a compiler. The
	// language is the key and the name of the image is the value.
	BuildImages map[string]string

	// RuntimeImages specifies the default container image with the
	// runtime for each language. This is the image that supplies the
	// environment for the test. The language is the key and the name
	// of the image is the value.
	RuntimeImages map[string]string
}
