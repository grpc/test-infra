package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// LoadTestGetter specifies methods for accessing or mutating load tests.
type LoadTestGetter interface {
	// Create saves a new test resource.
	Create(test *grpcv1.LoadTest, opts metav1.CreateOptions) (*grpcv1.LoadTest, error)

	// Get fetches a test, given its name and any options.
	Get(name string, opts metav1.GetOptions) (*grpcv1.LoadTest, error)

	// List fetches all tests, given its options.
	List(opts metav1.ListOptions) (*grpcv1.LoadTestList, error)

	// Delete removes a new test resource, given its name.
	Delete(name string, opts metav1.DeleteOptions) error
}

// LoadTestInterface provides methods for accessing a LoadTestGetter when given
// a namespace.
type LoadTestInterface interface {
	LoadTests(namespace string) LoadTestGetter
}
