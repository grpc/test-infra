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

package kubehelpers

import (
	"fmt"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
)

//IsPSMTest checks if a given LoadTest is a (proxied or proxyless) service
// mesh test. This test must be performed after validating the client specs.
func IsPSMTest(clients *[]grpcv1.Client) bool {
	for _, c := range *clients {
		if ContainerForName(config.XdsServerContainerName, c.Run) != nil {
			return true
		}
	}
	return false
}

// IsProxiedTest checks if the current test has a sidecar container specified.
// This check must be performed after validating the client specs.
func IsProxiedTest(clients *[]grpcv1.Client) bool {
	for _, c := range *clients {
		if ContainerForName(config.EnvoyContainerName, c.Run) != nil {
			return true
		}
	}
	return false
}

// IsClientsSpecValid checks if the given set of the client spec is valid.
func IsClientsSpecValid(clients *[]grpcv1.Client) (bool, error) {
	if len(*clients) == 0 {
		err := fmt.Errorf("no client specified")
		return false, err
	}
	var numberOfClientWithSidecar int
	var numberOfClientWithXdsServer int

	for _, c := range *clients {
		if ContainerForName(config.XdsServerContainerName, c.Run) != nil {
			numberOfClientWithXdsServer++
			if ContainerForName(config.EnvoyContainerName, c.Run) != nil {
				numberOfClientWithSidecar++
			}
		} else {
			if ContainerForName(config.EnvoyContainerName, c.Run) != nil {
				err := fmt.Errorf("encountered a client with envoy container but no xds-server container")
				return false, err
			}
		}
	}

	if numberOfClientWithXdsServer == 0 {
		return true, nil
	}

	if numberOfClientWithSidecar == 0 {
		if len(*clients) != numberOfClientWithXdsServer {
			err := fmt.Errorf("encountered some clients with xds-server container and some without xds-server container")
			return false, err
		}
	} else {
		if len(*clients) != numberOfClientWithSidecar {
			err := fmt.Errorf("encountered some clients with envoy container and some without envoy container")
			return false, err
		}
	}

	return true, nil
}
