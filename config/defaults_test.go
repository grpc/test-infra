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

package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

var _ = Describe("Defaults", func() {
	var defaults *Defaults

	BeforeEach(func() {
		defaults = &Defaults{
			ComponentNamespace: "component-default",
			DefaultPoolLabels: &PoolLabelMap{
				Client: "default-client-pool",
				Driver: "default-driver-pool",
				Server: "default-server-pool",
			},
			CloneImage:  "gcr.io/grpc-fake-project/test-infra/clone",
			ReadyImage:  "gcr.io/grpc-fake-project/test-infra/ready",
			DriverImage: "gcr.io/grpc-fake-project/test-infra/driver",
			Languages: []LanguageDefault{
				{
					Language:   "cxx",
					BuildImage: "l.gcr.io/google/bazel:latest",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/cxx",
				},
				{
					Language:   "go",
					BuildImage: "golang:1.20",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/go",
				},
				{
					Language:   "java",
					BuildImage: "java:jdk8",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/java",
				},
			},
			// KillAfter is the duration allowed for pods to respond after timeout, the value is in seconds.
			KillAfter: 20,
		}
	})

	Describe("Validate", func() {
		It("returns an error when missing the clone image", func() {
			defaults.CloneImage = ""
			err := defaults.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when missing the ready image", func() {
			defaults.ReadyImage = ""
			err := defaults.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when missing the driver image", func() {
			defaults.DriverImage = ""
			err := defaults.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when a language default lacks a name for a language", func() {
			defaults.Languages[1].Language = ""
			err := defaults.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when a language default lacks a build image", func() {
			defaults.Languages[1].BuildImage = ""
			err := defaults.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when a language default lacks a run image", func() {
			defaults.Languages[1].RunImage = ""
			err := defaults.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("returns nil for valid defaults", func() {
			err := defaults.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("SetLoadTestDefaults", func() {
		var loadtest *grpcv1.LoadTest
		var defaultImageMap *imageMap

		BeforeEach(func() {
			loadtest = completeLoadTest.DeepCopy()
			defaultImageMap = newImageMap(defaults.Languages)
		})

		Context("metadata", func() {
			It("sets default namespace when unset", func() {
				loadtest.Namespace = ""

				namespace := "foobar-buzz"
				defaults.ComponentNamespace = namespace

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(loadtest.Namespace).To(Equal(namespace))
			})

			It("does not override namespace when set", func() {
				namespace := "experimental"
				loadtest.Namespace = namespace

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(loadtest.Namespace).To(Equal(namespace))

			})
		})

		Context("driver", func() {
			var driver *grpcv1.Driver

			BeforeEach(func() {
				driver = loadtest.Spec.Driver
				Expect(driver).ToNot(BeNil())
			})

			It("sets default driver when nil", func() {
				loadtest.Spec.Driver = nil

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(loadtest.Spec.Driver).ToNot(BeNil())
			})

			It("does not override driver when set", func() {
				driver := new(grpcv1.Driver)
				loadtest.Spec.Driver = driver

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(loadtest.Spec.Driver).To(Equal(driver))
			})

			It("sets default name when unspecified", func() {
				driver.Name = nil
				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(driver.Name).ToNot(BeNil())
			})

			It("does not override pool when specified", func() {
				pool := "example-pool"
				driver.Pool = &pool

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(driver.Pool).ToNot(BeNil())
				Expect(*driver.Pool).To(Equal(pool))
			})

			It("sets missing image for clone init container", func() {
				repo := "https://github.com/grpc/grpc.git"
				gitRef := "master"

				driver.Clone = new(grpcv1.Clone)
				driver.Clone.Repo = &repo
				driver.Clone.GitRef = &gitRef
				driver.Clone.Image = nil

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(driver.Clone).ToNot(BeNil())
				Expect(driver.Clone.Image).ToNot(BeNil())
				Expect(*driver.Clone.Image).To(Equal(defaults.CloneImage))
			})

			It("sets missing image for build init container", func() {
				build := new(grpcv1.Build)
				build.Image = nil
				build.Command = []string{"bazel"}

				driver.Language = "cxx"
				driver.Build = build

				expectedBuildImage, err := defaultImageMap.buildImage(driver.Language)
				Expect(err).ToNot(HaveOccurred())

				err = defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(driver.Build).ToNot(BeNil())
				Expect(driver.Build.Image).ToNot(BeNil())
				Expect(*driver.Build.Image).To(Equal(expectedBuildImage))
			})

			It("errors if image for build init container cannot be inferred", func() {
				build := new(grpcv1.Build)
				build.Image = nil // no explicit image
				build.Command = []string{"make"}

				driver.Language = "fortran" // unknown language
				driver.Build = build

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).To(HaveOccurred())
			})

			It("does not error if build init container image cannot be inferred but is set", func() {
				image := "test-image"

				build := new(grpcv1.Build)
				build.Image = &image
				build.Command = []string{"make"}

				driver.Language = "fortran" // unknown language
				driver.Build = build

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

			})

			It("sets missing image for run container", func() {
				driver.Language = "cxx"
				driver.Run[0].Image = ""

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(driver.Run[0].Image).ToNot(BeEmpty())
				Expect(driver.Run[0].Image).To(Equal(defaults.DriverImage))
			})

			It("sets run container if the run container is nil", func() {
				driver.Run = nil
				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(driver.Run[0].Name).To(Equal("main"))
			})

			It("doesn't override the name of the first run container if set", func() {
				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(driver.Run[0].Name).To(Equal(loadtest.Spec.Driver.Run[0].Name))
			})

			It("doesn't override the run container image if set", func() {
				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(driver.Run[0].Image).ToNot(BeEmpty())
				Expect(driver.Run[0].Image).To(Equal(loadtest.Spec.Driver.Run[0].Image))
				Expect(driver.Run[0].Image).ToNot(Equal(defaults.DriverImage))
			})

			It("does not error if run container image cannot be inferred but is set", func() {
				image := "example-image"

				driver.Language = "fortran" // unknown language
				driver.Run[0].Image = image
				driver.Run[0].Command = []string{"do-stuff"}

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("server", func() {
			var server *grpcv1.Server

			BeforeEach(func() {
				server = &loadtest.Spec.Servers[0]
			})

			It("sets default name when unspecified", func() {
				server.Name = nil
				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(server.Name).ToNot(BeNil())
			})

			It("does not override pool when specified", func() {
				pool := "example-pool"
				server.Pool = &pool

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(server.Pool).ToNot(BeNil())
				Expect(*server.Pool).To(Equal(pool))
			})

			It("sets missing image for clone init container", func() {
				repo := "https://github.com/grpc/grpc.git"
				gitRef := "master"

				server.Clone = new(grpcv1.Clone)
				server.Clone.Repo = &repo
				server.Clone.GitRef = &gitRef
				server.Clone.Image = nil

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(server.Clone).ToNot(BeNil())
				Expect(server.Clone.Image).ToNot(BeNil())
				Expect(*server.Clone.Image).To(Equal(defaults.CloneImage))
			})

			It("sets missing image for build init container", func() {
				build := new(grpcv1.Build)
				build.Image = nil
				build.Command = []string{"bazel"}

				server.Language = "cxx"
				server.Build = build

				expectedBuildImage, err := defaultImageMap.buildImage(server.Language)
				Expect(err).ToNot(HaveOccurred())

				err = defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(server.Build).ToNot(BeNil())
				Expect(server.Build.Image).ToNot(BeNil())
				Expect(*server.Build.Image).To(Equal(expectedBuildImage))
			})

			It("errors if image for build init container cannot be inferred", func() {
				build := new(grpcv1.Build)
				build.Image = nil // no explicit image
				build.Command = []string{"make"}

				server.Language = "fortran" // unknown language
				server.Build = build

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).To(HaveOccurred())
			})

			It("does not error if build init container image cannot be inferred but is set", func() {
				image := "test-image"

				build := new(grpcv1.Build)
				build.Image = &image
				build.Command = []string{"make"}

				server.Language = "fortran" // unknown language
				server.Build = build

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

			})

			It("sets missing image for run container", func() {
				server.Language = "cxx"
				server.Run[0].Image = ""

				expectedRunImage, err := defaultImageMap.runImage(server.Language)
				Expect(err).ToNot(HaveOccurred())

				err = defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(server.Run[0].Image).ToNot(BeEmpty())
				Expect(server.Run[0].Image).To(Equal(expectedRunImage))
			})

			It("errors if image for run container cannot be inferred", func() {
				server.Build = nil // disable build to ensure run is the problem

				server.Language = "fortran" // unknown language
				server.Run[0].Image = ""    // no explicit image
				server.Run[0].Command = []string{"do-stuff"}

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).To(HaveOccurred())
			})

			It("does not error if run container image cannot be inferred but is set", func() {
				image := "example-image"

				server.Language = "fortran" // unknown language
				server.Run[0].Image = image
				server.Run[0].Command = []string{"do-stuff"}

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("client", func() {
			var client *grpcv1.Client

			BeforeEach(func() {
				client = &loadtest.Spec.Clients[0]
			})

			It("sets default name when unspecified", func() {
				client.Name = nil
				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.Name).ToNot(BeNil())
			})

			It("does not override pool when specified", func() {
				pool := "example-pool"
				client.Pool = &pool

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.Pool).ToNot(BeNil())
				Expect(*client.Pool).To(Equal(pool))
			})

			It("sets missing image for clone init container", func() {
				repo := "https://github.com/grpc/grpc.git"
				gitRef := "master"

				client.Clone = new(grpcv1.Clone)
				client.Clone.Repo = &repo
				client.Clone.GitRef = &gitRef
				client.Clone.Image = nil

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.Clone).ToNot(BeNil())
				Expect(client.Clone.Image).ToNot(BeNil())
				Expect(*client.Clone.Image).To(Equal(defaults.CloneImage))
			})

			It("sets missing image for build init container", func() {
				build := new(grpcv1.Build)
				build.Image = nil
				build.Command = []string{"bazel"}

				client.Language = "cxx"
				client.Build = build

				expectedBuildImage, err := defaultImageMap.buildImage(client.Language)
				Expect(err).ToNot(HaveOccurred())

				err = defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.Build).ToNot(BeNil())
				Expect(client.Build.Image).ToNot(BeNil())
				Expect(*client.Build.Image).To(Equal(expectedBuildImage))
			})

			It("errors if image for build init container cannot be inferred", func() {
				build := new(grpcv1.Build)
				build.Image = nil // no explicit image
				build.Command = []string{"make"}

				client.Language = "fortran" // unknown language
				client.Build = build

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).To(HaveOccurred())
			})

			It("does not error if build init container image cannot be inferred but is set", func() {
				image := "test-image"

				build := new(grpcv1.Build)
				build.Image = &image
				build.Command = []string{"make"}

				client.Language = "fortran" // unknown language
				client.Build = build

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

			})

			It("sets missing image for run container", func() {
				client.Language = "cxx"
				client.Run[0].Image = ""

				expectedRunImage, err := defaultImageMap.runImage(client.Language)
				Expect(err).ToNot(HaveOccurred())

				err = defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.Run[0].Image).ToNot(BeEmpty())
				Expect(client.Run[0].Image).To(Equal(expectedRunImage))
			})

			It("errors if image for run container cannot be inferred", func() {
				client.Language = "fortran" // unknown language
				client.Run[0].Image = ""    // no explicit image
				client.Run[0].Command = []string{"do-stuff"}

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).To(HaveOccurred())
			})

			It("does not error if run container image cannot be inferred but is set", func() {
				image := "example-image"

				client.Language = "fortran" // unknown language
				client.Run[0].Image = image
				client.Run[0].Command = []string{"do-stuff"}

				err := defaults.SetLoadTestDefaults(loadtest)
				Expect(err).ToNot(HaveOccurred())
			})
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
				Name:     &driverComponentName,
				Language: "cxx",
				Pool:     &driverPool,
				Run: []corev1.Container{{
					Image: driverImage,
				}},
			},

			Servers: []grpcv1.Server{
				{
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
					Run: []corev1.Container{{
						Image:   runImage,
						Command: runCommand,
						Args:    serverRunArgs,
					}},
				},
			},

			Clients: []grpcv1.Client{
				{
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
					Run: []corev1.Container{{
						Image:   runImage,
						Command: runCommand,
						Args:    clientRunArgs,
					}},
				},
			},

			Results: &grpcv1.Results{
				BigQueryTable: &bigQueryTable,
			},

			ScenariosJSON: "{\"scenarios\": []}",
		},
	}
}()
