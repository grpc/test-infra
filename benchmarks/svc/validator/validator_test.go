// Copyright 2020 gRPC authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"testing"

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

func TestValidatorValidate(t *testing.T) {
	t.Run("driver", func(t *testing.T) {
		cases := []struct {
			driverNil   bool
			shouldError bool
		}{
			{driverNil: true, shouldError: true},
			{driverNil: false, shouldError: false},
		}

		for _, tc := range cases {
			description := "missing"
			if !tc.driverNil {
				description = "present"
			}

			t.Run(description, func(t *testing.T) {
				validator := &Validator{}

				var driver *types.Component
				if !tc.driverNil {
					driver = types.NewComponent("driver-image", types.DriverComponent)
				}

				workers := []*types.Component{
					types.NewComponent("server-image", types.ServerComponent),
					types.NewComponent("client-image", types.ClientComponent),
				}

				// TODO: Replace <nil> scenario when scenario validations are added
				session := types.NewSession(driver, workers, nil)

				err := validator.Validate(session)

				if tc.shouldError && err == nil {
					t.Fatalf("did not error")
				} else if !tc.shouldError && err != nil {
					t.Fatalf("returned unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("image name prefix", func(t *testing.T) {
		cases := []struct {
			description  string
			prefix       string
			driverImage  string
			serverImage  string
			clientImages []string
			shouldError  bool
		}{
			{
				description: "match",
				prefix:      "gcr.io/grpc-fake",
				driverImage: "gcr.io/grpc-fake/valid-image",
				serverImage: "gcr.io/grpc-fake/valid-image",
				clientImages: []string{
					"gcr.io/grpc-fake/valid-image",
					"gcr.io/grpc-fake/valid-image",
				},
				shouldError: false,
			},
			{
				description: "mismatch in driver",
				prefix:      "gcr.io/grpc-fake",
				driverImage: "gcr.io/grpc-fak/invalid-image",
				serverImage: "gcr.io/grpc-fake/valid-image",
				clientImages: []string{
					"gcr.io/grpc-fake/valid-image",
					"gcr.io/grpc-fake/valid-image",
				},
				shouldError: true,
			},
			{
				description: "mismatch in server",
				prefix:      "gcr.io/grpc-fake",
				driverImage: "gcr.io/grpc-fake/valid-image",
				serverImage: "gcr.io/grpc-fak/invalid-image",
				clientImages: []string{
					"gcr.io/grpc-fake/valid-image",
					"gcr.io/grpc-fake/valid-image",
				},
				shouldError: true,
			},
			{
				description: "mismatch in client 1",
				prefix:      "gcr.io/grpc-fake",
				driverImage: "gcr.io/grpc-fake/valid-image",
				serverImage: "gcr.io/grpc-fake/valid-image",
				clientImages: []string{
					"gcr.io/grpc-fak/invalid-image",
					"gcr.io/grpc-fake/valid-image",
				},
				shouldError: true,
			},
			{
				description: "mismatch in client 2",
				prefix:      "gcr.io/grpc-fake",
				driverImage: "gcr.io/grpc-fake/valid-image",
				serverImage: "gcr.io/grpc-fake/valid-image",
				clientImages: []string{
					"gcr.io/grpc-fake/valid-image",
					"gcr.io/grpc-fak/invalid-image",
				},
				shouldError: true,
			},
		}

		for _, tc := range cases {
			t.Run(tc.description, func(t *testing.T) {
				validator := &Validator{ImageNamePrefix: tc.prefix}

				var workers []*types.Component
				workers = append(workers, types.NewComponent(tc.serverImage, types.ServerComponent))
				for _, clientImage := range tc.clientImages {
					workers = append(workers, types.NewComponent(clientImage, types.ClientComponent))
				}
				driver := types.NewComponent(tc.driverImage, types.DriverComponent)

				// TODO: Replace <nil> scenario when scenario validations are added
				session := types.NewSession(driver, workers, nil)

				err := validator.Validate(session)

				if tc.shouldError && err == nil {
					t.Fatalf("did not error")
				} else if !tc.shouldError && err != nil {
					t.Fatalf("returned unexpected error: %v", err)
				}
			})
		}
	})
}
