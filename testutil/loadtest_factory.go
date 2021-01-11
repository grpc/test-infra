package testutil

import (
	"fmt"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/optional"
)

// NewLoadTest creates a LoadTest instance for unit and e2e tests. It accepts
// the number of servers and clients that should be included in the test. Each
// server and client will have the word "server-" or "client-" as a prefix and
// an incrementing number as the suffix. For example, 3 servers will have the
// names: "server-1", "server-2" and "server-3".
func NewLoadTest(serverCount, clientCount int) *grpcv1.LoadTest {
	test := &grpcv1.LoadTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid.New().String(),
			Namespace: corev1.NamespaceDefault,
		},
		Spec: grpcv1.LoadTestSpec{
			TimeoutSeconds: 300,
			TTLSeconds:     600,
			Driver: &grpcv1.Driver{
				Name:     optional.StringPtr("driver"),
				Language: "cxx",
				Pool:     optional.StringPtr("test-pool"),
				Run: grpcv1.Run{
					Image: optional.StringPtr("gcr.io/grpc-test-example/driver:v1"),
				},
			},
			Results: &grpcv1.Results{
				BigQueryTable: optional.StringPtr("example-dataset.example-table"),
			},
			ScenariosJSON: "{\"scenarios\": []}",
		},
	}

	for i := 1; i <= serverCount; i++ {
		test.Spec.Servers = append(test.Spec.Servers, grpcv1.Server{
			Name:     optional.StringPtr(fmt.Sprintf("server-%d", i)),
			Language: "go",
			Pool:     optional.StringPtr("test-pool"),
			Clone: &grpcv1.Clone{
				Image:  optional.StringPtr("gcr.io/grpc-test-example/clone:v1"),
				Repo:   optional.StringPtr("https://github.com/grpc/test-infra.git"),
				GitRef: optional.StringPtr("master"),
			},
			Build: &grpcv1.Build{
				Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
				Command: []string{"go"},
				Args:    []string{"build", "-o", "server", "./server/main.go"},
			},
			Run: grpcv1.Run{
				Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
				Command: []string{"./server"},
				Args:    []string{"-verbose"},
			},
		})
	}

	for i := 1; i <= clientCount; i++ {
		test.Spec.Clients = append(test.Spec.Clients, grpcv1.Client{
			Name:     optional.StringPtr(fmt.Sprintf("client-%d", i)),
			Language: "go",
			Pool:     optional.StringPtr("test-pool"),
			Clone: &grpcv1.Clone{
				Image:  optional.StringPtr("gcr.io/grpc-test-example/clone:v1"),
				Repo:   optional.StringPtr("https://github.com/grpc/test-infra.git"),
				GitRef: optional.StringPtr("master"),
			},
			Build: &grpcv1.Build{
				Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
				Command: []string{"go"},
				Args:    []string{"build", "-o", "client", "./client/main.go"},
			},
			Run: grpcv1.Run{
				Image:   optional.StringPtr("gcr.io/grpc-test-example/go:v1"),
				Command: []string{"./client"},
				Args:    []string{"-verbose"},
			},
		})
	}

	return test
}
