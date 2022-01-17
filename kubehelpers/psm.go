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

// IsPSMTest checks if a given Loadtest is a PSM test. All clients running
// a scenario should have the same configuration otherwise a error will be
// returned.
func IsPSMTest(clients *[]grpcv1.Client) (bool, error) {

	if len(*clients) == 0 {
		err := fmt.Errorf("no client specified in the given load test")
		return false, err
	}

	psmClient := 0
	for _, c := range *clients {
		if c.XDS != nil {
			psmClient++
		}
	}
	if psmClient != 0 && len(*clients) != psmClient {
		err := fmt.Errorf("client running the same test should all be the same type: the should all have xds container specified or none of them should have xds server container specified")
		return true, err
	}

	return psmClient != 0, nil
}

// IsProxiedTest is to check if the current test have sidecar container specified.
func IsProxiedTest(clients *[]grpcv1.Client) (bool, error) {
	if len(*clients) == 0 {
		err := fmt.Errorf("no client specified in the given load test")
		return false, err
	}

	proxiedClient := 0
	for _, c := range *clients {
		if c.Sidecar != nil {
			proxiedClient++
			if c.XDS == nil {
				err := fmt.Errorf("encountered a client that has sidecar container but no xds container")
				return true, err
			}
		}
	}
	if proxiedClient != 0 && len(*clients) != proxiedClient {
		err := fmt.Errorf("client running the same test should all be the same type: the should all have sidecar container specified or none of them should have sidecar container specified")
		return true, err
	}

	return proxiedClient != 0, nil
}
