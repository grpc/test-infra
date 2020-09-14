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

package config

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
