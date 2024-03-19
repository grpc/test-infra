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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/optional"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("IsPSMTest", func() {
	var clients *[]grpcv1.Client

	It("returns true for a client set that at least one client has xds container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-1"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					},
				},
			},
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual := IsPSMTest(clients)
		Expect(actual).To(BeTrue())
	})

	It("returns false for a client set that none of the client has xds container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-1"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					},
				},
			},
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual := IsPSMTest(clients)
		Expect(actual).To(BeFalse())
	})
})

var _ = Describe("IsProxiedTest", func() {
	var clients *[]grpcv1.Client

	It("returns true for a client set that at least one client has sidecar container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-1"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "sidecar",
						Image:   "gcr.io/grpc-test-example/sidecar:v1",
						Command: []string{"./sidecar"},
						Args:    []string{"-verbose"},
					},
				},
			},
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "sidecar",
						Image:   "gcr.io/grpc-test-example/sidecar:v1",
						Command: []string{"./sidecar"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual := IsProxiedTest(clients)
		Expect(actual).To(BeTrue())
	})

	It("returns false for a client set that none of the client has sidecar container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-1"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					},
				},
			},
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual := IsProxiedTest(clients)
		Expect(actual).To(BeFalse())
	})
})

var _ = Describe("IsClientsSpecValid", func() {
	var clients *[]grpcv1.Client

	It("returns false and an error with an empty client set", func() {
		clients = &[]grpcv1.Client{}
		actual, err := IsClientsSpecValid(clients)
		Expect(actual).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})

	It("returns false and an error for a client set that only some of the clients have xds container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-1"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					},
				},
			},
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual, err := IsClientsSpecValid(clients)
		Expect(actual).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})
	It("returns false and an error for a client set that only some of the clients have sidecar container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-1"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					},
				},
			},
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "sidecar",
						Image:   "gcr.io/grpc-test-example/sidecar:v1",
						Command: []string{"./sidecar"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual, err := IsClientsSpecValid(clients)
		Expect(actual).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})
	It("returns false and an error for a client set that at lease one of the client have sidecar container but no xds container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "sidecar",
						Image:   "gcr.io/grpc-test-example/sidecar:v1",
						Command: []string{"./sidecar"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual, err := IsClientsSpecValid(clients)
		Expect(actual).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})
	It("returns true and nil for a client set that all clients only xds container", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual, err := IsClientsSpecValid(clients)
		Expect(actual).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())
	})
	It("returns true and nil for a client set that all clients have xds server and sidecar containers", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{
					{
						Image:   "gcr.io/grpc-test-example/go:v1",
						Command: []string{"./client"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "xds-server",
						Image:   "gcr.io/grpc-test-example/xds:v1",
						Command: []string{"./xds"},
						Args:    []string{"-verbose"},
					}, {
						Name:    "sidecar",
						Image:   "gcr.io/grpc-test-example/sidecar:v1",
						Command: []string{"./sidecar"},
						Args:    []string{"-verbose"},
					},
				},
			},
		}
		actual, err := IsClientsSpecValid(clients)
		Expect(actual).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())
	})
	It("returns true and nil for a client set that all clients are running regular test", func() {
		clients = &[]grpcv1.Client{
			{
				Name:     optional.StringPtr("client-2"),
				Language: "go",
				Pool:     optional.StringPtr("workers-a"),
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
				Run: []corev1.Container{{
					Image:   "gcr.io/grpc-test-example/go:v1",
					Command: []string{"./client"},
					Args:    []string{"-verbose"},
				}},
			},
		}
		actual, err := IsClientsSpecValid(clients)
		Expect(actual).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())
	})
})
