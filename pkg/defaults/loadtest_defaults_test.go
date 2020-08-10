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

package defaults

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

func newLoadTest(name string) *grpcv1.LoadTest {
	return &grpcv1.LoadTest{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

var _ = Describe("CopyWithDefaults", func() {
	var loadtest *grpcv1.LoadTest
	var defaultOptions *Defaults
	var defaultImageMap *imageMap

	BeforeEach(func() {
		loadtest = completeLoadTest.DeepCopy()

		defaultOptions = &Defaults{
			DriverPool:  "drivers",
			WorkerPool:  "workers-8core",
			DriverPort:  10000,
			ServerPort:  10010,
			CloneImage:  "gcr.io/grpc-fake-project/test-infra/clone",
			DriverImage: "gcr.io/grpc-fake-project/test-infra/driver",
			Languages: []LanguageDefault{
				{
					Language:   "cxx",
					BuildImage: "l.gcr.io/google/bazel:latest",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/cxx",
				},
				{
					Language:   "go",
					BuildImage: "golang:1.14",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/go",
				},
				{
					Language:   "java",
					BuildImage: "java:jdk8",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/java",
				},
			},
		}

		defaultImageMap = newImageMap(defaultOptions.Languages)
	})

	Context("driver", func() {
		It("sets default name when unspecified", func() {
			loadtest.Spec.Driver.Name = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(copy.Spec.Driver.Name).ToNot(BeNil())
		})

		It("sets default when nil", func() {
			loadtest.Spec.Driver = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(copy.Spec.Driver).ToNot(BeNil())
			Expect(*copy.Spec.Driver.Run.Image).To(Equal(defaultOptions.DriverImage))
		})

		It("does not override when set", func() {
			driverImage := "gcr.io/grpc-example/test-image"
			loadtest.Spec.Driver.Run.Image = &driverImage
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Run.Image).To(Equal(driverImage))
		})

		It("sets default driver pool when nil", func() {
			loadtest.Spec.Driver.Pool = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Pool).To(Equal(defaultOptions.DriverPool))
		})

		It("does not override pool when set", func() {
			testPool := "preset-pool"
			loadtest.Spec.Driver.Pool = &testPool
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Pool).To(Equal(testPool))
		})

		It("sets clone image when missing", func() {
			repo := "https://github.com/grpc/grpc.git"
			gitRef := "master"

			clone := new(grpcv1.Clone)
			clone.Image = nil
			clone.Repo = &repo
			clone.GitRef = &gitRef

			loadtest.Spec.Driver.Clone = clone

			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Clone.Image).To(Equal(defaultOptions.CloneImage))
		})

		It("sets build image when missing", func() {
			build := new(grpcv1.Build)
			build.Image = nil
			build.Command = []string{"bazel"}

			loadtest.Spec.Driver.Build = build

			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			expectedBuildImage, _ := defaultImageMap.buildImage("cxx")
			Expect(*copy.Spec.Driver.Build.Image).To(Equal(expectedBuildImage))
		})

		It("errors when build image missing and language unknown", func() {
			build := new(grpcv1.Build)
			build.Image = nil

			loadtest.Spec.Driver.Language = "fortran"
			loadtest.Spec.Driver.Build = build

			_, err := CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("servers", func() {
		It("sets default name when unspecified", func() {
			loadtest.Spec.Servers[0].Name = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(copy.Spec.Servers[0].Name).ToNot(BeNil())
		})

		It("sets default name that is unique", func() {
			server2 := loadtest.Spec.Servers[0].DeepCopy()
			loadtest.Spec.Servers = append(loadtest.Spec.Servers, *server2)

			loadtest.Spec.Servers[0].Name = nil
			loadtest.Spec.Servers[1].Name = nil

			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(copy.Spec.Servers[0].Name).ToNot(Equal(copy.Spec.Servers[1].Name))
		})

		It("sets clone image when missing", func() {
			loadtest.Spec.Servers[0].Clone.Image = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Servers[0].Clone.Image).To(Equal(defaultOptions.CloneImage))
		})

		It("sets build image when missing", func() {
			loadtest.Spec.Servers[0].Build.Image = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			expectedBuildImage, _ := defaultImageMap.buildImage("cxx")
			Expect(*copy.Spec.Servers[0].Build.Image).To(Equal(expectedBuildImage))
		})

		It("sets run image when missing", func() {
			loadtest.Spec.Servers[0].Run.Image = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			expectedRunImage, _ := defaultImageMap.runImage("cxx")
			Expect(*copy.Spec.Servers[0].Run.Image).To(Equal(expectedRunImage))
		})

		It("does not override pool when set", func() {
			pool := "custom-pool"
			loadtest.Spec.Servers[0].Pool = &pool
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Servers[0].Pool).To(Equal(pool))
		})

		It("sets default worker pool when nil", func() {
			loadtest.Spec.Servers[0].Pool = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Servers[0].Pool).To(Equal(defaultOptions.WorkerPool))
		})

		It("errors when run image missing and language unknown", func() {
			loadtest.Spec.Servers[0].Language = "fortran"
			loadtest.Spec.Servers[0].Run.Image = nil
			_, err := CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})

		It("errors when build image missing and language unknown", func() {
			loadtest.Spec.Servers[0].Language = "fortran"
			loadtest.Spec.Servers[0].Build.Image = nil
			_, err := CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("clients", func() {
		It("sets default name when unspecified", func() {
			loadtest.Spec.Clients[0].Name = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(copy.Spec.Clients[0].Name).ToNot(BeNil())
		})

		It("sets default name that is unique", func() {
			client2 := loadtest.Spec.Clients[0].DeepCopy()
			loadtest.Spec.Clients = append(loadtest.Spec.Clients, *client2)

			loadtest.Spec.Clients[0].Name = nil
			loadtest.Spec.Clients[1].Name = nil

			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(copy.Spec.Clients[0].Name).ToNot(Equal(copy.Spec.Clients[1].Name))
		})

		It("sets clone image when missing", func() {
			loadtest.Spec.Clients[0].Clone.Image = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Clients[0].Clone.Image).To(Equal(defaultOptions.CloneImage))
		})

		It("sets build image when missing", func() {
			loadtest.Spec.Clients[0].Build.Image = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			expectedBuildImage, _ := defaultImageMap.buildImage("cxx")
			Expect(*copy.Spec.Clients[0].Build.Image).To(Equal(expectedBuildImage))
		})

		It("sets run image when missing", func() {
			loadtest.Spec.Clients[0].Run.Image = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			expectedRunImage, _ := defaultImageMap.runImage("cxx")
			Expect(*copy.Spec.Clients[0].Run.Image).To(Equal(expectedRunImage))
		})

		It("does not override pool when set", func() {
			pool := "custom-pool"
			loadtest.Spec.Clients[0].Pool = &pool
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Clients[0].Pool).To(Equal(pool))
		})

		It("sets default worker pool when nil", func() {
			loadtest.Spec.Clients[0].Pool = nil
			copy, _ := CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Clients[0].Pool).To(Equal(defaultOptions.WorkerPool))
		})

		It("errors when run image missing and language unknown", func() {
			loadtest.Spec.Clients[0].Language = "fortran"
			loadtest.Spec.Clients[0].Run.Image = nil
			_, err := CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})

		It("errors when build image missing and language unknown", func() {
			loadtest.Spec.Clients[0].Language = "fortran"
			loadtest.Spec.Clients[0].Build.Image = nil
			_, err := CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})
	})
})

var completeLoadTest = func() *grpcv1.LoadTest {
	cloneImage := "docker.pkg.github.com/grpc/test-infra/clone"
	cloneRepo := "https://github.com/grpc/grpc.git"
	cloneGitRef := "master"

	buildImage := "l.gcr.io/google/bazel:latest"
	buildCommand := []string{"bazel"}
	buildArgs := []string{"build", "//test/cpp/qps:qps_worker"}

	driverImage := "docker.pkg.github.com/grpc/test-infra/driver"
	runImage := "docker.pkg.github.com/grpc/test-infra/cxx"
	runCommand := []string{"bazel-bin/test/cpp/qps/qps_worker"}
	clientRunArgs := []string{"--driver_port=10000"}
	serverRunArgs := append(clientRunArgs, "--server_port=10010")

	bigQueryTable := "grpc-testing.e2e_benchmark.foobarbuzz"

	driverPool := "drivers"
	workerPool := "workers-8core"

	driverComponentName := "driver"
	serverComponentName := "server"
	clientComponentName := "client-1"

	return &grpcv1.LoadTest{
		Spec: grpcv1.LoadTestSpec{
			Driver: &grpcv1.Driver{
				Component: grpcv1.Component{
					Name:     &driverComponentName,
					Language: "cxx",
					Pool:     &driverPool,
					Run: grpcv1.Run{
						Image: &driverImage,
					},
				},
			},

			Servers: []grpcv1.Server{
				{
					Component: grpcv1.Component{
						Name:     &serverComponentName,
						Language: "cxx",
						Pool:     &workerPool,
						Clone: &grpcv1.Clone{
							Image:  &cloneImage,
							Repo:   &cloneRepo,
							GitRef: &cloneGitRef,
						},
						Build: &grpcv1.Build{
							Image:   &buildImage,
							Command: buildCommand,
							Args:    buildArgs,
						},
						Run: grpcv1.Run{
							Image:   &runImage,
							Command: runCommand,
							Args:    serverRunArgs,
						},
					},
				},
			},

			Clients: []grpcv1.Client{
				{
					Component: grpcv1.Component{
						Name:     &clientComponentName,
						Language: "cxx",
						Pool:     &workerPool,
						Clone: &grpcv1.Clone{
							Image:  &cloneImage,
							Repo:   &cloneRepo,
							GitRef: &cloneGitRef,
						},
						Build: &grpcv1.Build{
							Image:   &buildImage,
							Command: buildCommand,
							Args:    buildArgs,
						},
						Run: grpcv1.Run{
							Image:   &runImage,
							Command: runCommand,
							Args:    clientRunArgs,
						},
					},
				},
			},

			Results: &grpcv1.Results{
				BigQueryTable: &bigQueryTable,
			},

			Scenarios: []grpcv1.Scenario{
				{Name: "cpp-example-scenario"},
			},
		},
	}
}()
