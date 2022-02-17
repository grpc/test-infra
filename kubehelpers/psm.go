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

package kubehelpers

import (
	"fmt"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// IsPSMTest checks if a given Loadtest is a PSM test. This check must be
// used after validate the spec of the clients.
func IsPSMTest(clients *[]grpcv1.Client) bool {
	for _, c := range *clients {
		if ContainerForName("xds-server", c.Run) != nil {
			return true
		}
	}
	return false
}

// IsProxiedTest is to check if the current test have sidecar container
// specified. This check must be used after validate the spec of the clients.
func IsProxiedTest(clients *[]grpcv1.Client) bool {
	for _, c := range *clients {
		if ContainerForName("envoy", c.Run) != nil {
			return true
		}
	}
	return false
}

// IsClientsSpecValid checks if the given set of the client specs are valid.
func IsClientsSpecValid(clients *[]grpcv1.Client) (bool, error) {
	if len(*clients) == 0 {
		err := fmt.Errorf("no client specified in the given load test")
		return false, err
	}
	var numberOfClientWithSidecar int
	var numberOfClientWithXdsServer int

	for _, c := range *clients {
		if ContainerForName("xds-server", c.Run) != nil {
			numberOfClientWithXdsServer++
			if ContainerForName("envoy", c.Run) != nil {
				numberOfClientWithSidecar++
			}
		} else {
			if ContainerForName("envoy", c.Run) != nil {
				err := fmt.Errorf("encountered a client specified a sidecar container without specifyling xds-server container")
				return false, err
			}
		}
	}

	if numberOfClientWithXdsServer == 0 {
		return true, nil
	}

	if numberOfClientWithSidecar == 0 {
		if len(*clients) != numberOfClientWithXdsServer {
			err := fmt.Errorf("only some of the clients have xds-server container specified")
			return false, err
		}
	} else {
		if len(*clients) != numberOfClientWithSidecar {
			err := fmt.Errorf("only some of the clients have envoy container specified")
			return false, err
		}
	}

	return true, nil
}
