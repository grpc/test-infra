/*
Copyright 2022 gRPC authors.

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
	// ServerUpdatePort is the port on the xDS server to listen to
	// configuration for PSM test only.
	ServerUpdatePort = 18005

	// XdsServerContainerName holds the name of the xds-server
	// container for PSM test only.
	XdsServerContainerName = "xds-server"

	// SidecarContainerName holds the name of the sidecar
	// container for a proxied PSM test only.
	SidecarContainerName = "sidecar"
)
