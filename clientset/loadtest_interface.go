/*
Copyright 2021 gRPC authors.

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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// LoadTestGetter specifies methods for accessing or mutating load tests.
type LoadTestGetter interface {
	// Create saves a new test resource.
	Create(ctx context.Context, test *grpcv1.LoadTest, opts metav1.CreateOptions) (*grpcv1.LoadTest, error)

	// Get fetches a test, given its name and any options.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*grpcv1.LoadTest, error)

	// List fetches all tests, given its options.
	List(ctx context.Context, opts metav1.ListOptions) (*grpcv1.LoadTestList, error)

	// Delete removes a new test resource, given its name.
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// LoadTestInterface provides methods for accessing a LoadTestGetter when given
// a namespace.
type LoadTestInterface interface {
	LoadTests(namespace string) LoadTestGetter
}
