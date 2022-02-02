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
func IsPSMTest(clients *[]grpcv1.Client) (bool, error) {
	for _, c := range *clients {
		if c.XDSServer != nil {
			return true, nil
		}
	}
	return false, nil
}

// IsProxiedTest is to check if the current test have sidecar container
//specified. This check must be used after validate the spec of the clients.
func IsProxiedTest(clients *[]grpcv1.Client) (bool, error) {
	for _, c := range *clients {
		if c.Sidecar != nil {
			return true, nil
		}
	}
	return false, nil
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
		if c.XDSServer != nil {
			numberOfClientWithXdsServer++
			if c.Sidecar != nil {
				numberOfClientWithSidecar++
			}
		} else {
			if c.Sidecar != nil {
				err := fmt.Errorf("client specified a sidecar container without specifyling xdsServer container")
				return false, err
			}
		}
	}

	// no client have xdsServer container specified, running a regular test.
	if numberOfClientWithXdsServer == 0 {
		return true, nil
	}

	if numberOfClientWithSidecar == 0 {
		// running a proxyless test
		if len(*clients) != numberOfClientWithXdsServer {
			err := fmt.Errorf("client running the same test should all be the same type: the should all have xds container specified or none of them should have xds server container specified")
			return false, err
		}
	} else {
		if len(*clients) != numberOfClientWithSidecar {
			err := fmt.Errorf("client running the same test should all be the same type: the should all have sidecar container specified or none of them should have sidecar container specified")
			return false, err
		}
	}

	return true, nil
}
