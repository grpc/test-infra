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

package podbuilder

import (
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/kubehelpers"
	"github.com/grpc/test-infra/optional"
)

// getNames accepts a slice of objects with a Name field. It returns the names
// of all objects in the slice as a slice of strings. The function
// will panic when the slice parameter is not a slice of objects with a Name
// field.
var getNames = func(slice interface{}) []string {
	var names []string
	list := reflect.ValueOf(slice)

	for i := 0; i < list.Len(); i++ {
		item := list.Index(i)
		name := item.FieldByName("Name")
		names = append(names, name.String())
	}

	return names
}

// getValue accepts the name of a desired object, the field to return and a
// slice of objects to search through. The slice should only be composed of
// objects with a Name field, which is used for the comparison. The function
// will panic when the field does not exist or slice is not a slice of objects
// with a Name field. If no matching object is found but the slice and field are
// valid, a nil pointer is returned.
var getValue = func(name, field string, slice interface{}) interface{} {
	list := reflect.ValueOf(slice)

	for i := 0; i < list.Len(); i++ {
		item := list.Index(i)

		if name == item.FieldByName("Name").String() {
			return item.FieldByName(field).Interface()
		}
	}

	return nil
}

var _ = Describe("PodBuilder", func() {
	var test *grpcv1.LoadTest
	var testSpec *grpcv1.LoadTestSpec
	var defaults *config.Defaults
	var builder *PodBuilder

	BeforeEach(func() {
		test = newLoadTest()
		testSpec = &test.Spec
		defaults = newDefaults()
		builder = New(defaults, test)
	})

	Describe("PodForClient", func() {
		var client *grpcv1.Client

		BeforeEach(func() {
			client = &testSpec.Clients[0]
		})

		It("sets the namespace to match the test", func() {
			pod, err := builder.PodForClient(client)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Namespace).To(Equal(test.Namespace))
		})

		It("sets a label with the name of the load test", func() {
			pod, err := builder.PodForClient(client)
			Expect(err).ToNot(HaveOccurred())

			testName, ok := pod.ObjectMeta.Labels[config.LoadTestLabel]
			Expect(ok).To(BeTrue())
			Expect(testName).To(Equal(test.Name))
		})

		It("sets a label indicating it is a client", func() {
			pod, err := builder.PodForClient(client)
			Expect(err).ToNot(HaveOccurred())

			role, ok := pod.ObjectMeta.Labels[config.RoleLabel]
			Expect(ok).To(BeTrue())
			Expect(role).To(Equal(config.ClientRole))
		})

		It("sets a label with the name of the client", func() {
			pod, err := builder.PodForClient(client)
			Expect(err).ToNot(HaveOccurred())

			componentName, ok := pod.ObjectMeta.Labels[config.ComponentNameLabel]
			Expect(ok).To(BeTrue())
			Expect(componentName).To(Equal(*client.Name))
		})

		It("sets node selector to match pool", func() {
			client.Pool = optional.StringPtr("testing-pool")

			pod, err := builder.PodForClient(client)
			Expect(err).ToNot(HaveOccurred())

			Expect(pod.Spec.NodeSelector).ToNot(BeNil())
			Expect(pod.Spec.NodeSelector["pool"]).To(Equal(*client.Pool))
		})

		Context("clone init container", func() {
			It("contains an init container named clone when clone instructions are present", func() {
				client.Clone = new(grpcv1.Clone)
				client.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				client.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.CloneInitContainerName))
			})

			It("does not contain an init container named clone when clone instructions are not present", func() {
				client.Clone = nil

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).ToNot(ContainElement(config.CloneInitContainerName))
			})

			It("sets an environment variable with the git repository", func() {
				client.Clone = new(grpcv1.Clone)
				client.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				client.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)

				var gitRepoEnv *corev1.EnvVar
				for i := range cloneContainer.Env {
					env := &cloneContainer.Env[i]

					if env.Name == config.CloneRepoEnv {
						gitRepoEnv = env
					}
				}

				Expect(gitRepoEnv).ToNot(BeNil())
				Expect(gitRepoEnv.Value).To(Equal(*client.Clone.Repo))
			})

			It("sets an environment variable with the git ref", func() {
				client.Clone = new(grpcv1.Clone)
				client.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				client.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)

				var gitRefEnv *corev1.EnvVar
				for i := range cloneContainer.Env {
					env := &cloneContainer.Env[i]

					if env.Name == config.CloneGitRefEnv {
						gitRefEnv = env
					}
				}

				Expect(gitRefEnv).ToNot(BeNil())
				Expect(gitRefEnv.Value).To(Equal(*client.Clone.GitRef))
			})

			It("creates volume mount for workspace", func() {
				client.Clone = new(grpcv1.Clone)
				client.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				client.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)
				Expect(cloneContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(cloneContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})
		})

		Context("build init container", func() {
			It("contains a init container named build when build instructions are present", func() {
				client.Build = new(grpcv1.Build)
				client.Build.Command = []string{"go"}
				client.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.BuildInitContainerName))
			})

			It("does not contain an init container named build when build instructions are not present", func() {
				client.Build = nil

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).ToNot(ContainElement(config.BuildInitContainerName))
			})

			It("sets working directory to workspace", func() {
				client.Build = new(grpcv1.Build)
				client.Build.Command = []string{"go"}
				client.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.BuildInitContainerName))

				buildContainer := kubehelpers.ContainerForName(config.BuildInitContainerName, pod.Spec.InitContainers)
				Expect(buildContainer.WorkingDir).To(Equal(config.WorkspaceMountPath))
			})

			It("creates volume mount for workspace", func() {
				client.Build = new(grpcv1.Build)
				client.Build.Command = []string{"go"}
				client.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				buildContainer := kubehelpers.ContainerForName(config.BuildInitContainerName, pod.Spec.InitContainers)
				Expect(buildContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(buildContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})
		})

		Context("run container", func() {
			It("creates volume mount for workspace", func() {
				client.Run = grpcv1.Run{}
				client.Run.Command = []string{"go"}
				client.Run.Args = []string{"run", "main.go"}

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers).ToNot(BeEmpty())

				runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
				Expect(runContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(runContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})

			It("exposes the driver port", func() {
				client.Run = grpcv1.Run{}
				client.Run.Command = []string{"go"}
				client.Run.Args = []string{"run", "main.go"}

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers).ToNot(BeEmpty())

				runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
				Expect(getNames(runContainer.Ports)).To(ContainElement("driver"))
				Expect(getValue("driver", "ContainerPort", runContainer.Ports)).To(BeEquivalentTo(config.DriverPort))
			})

			It("appends the driver port command line argument", func() {
				client.Run = grpcv1.Run{}
				client.Run.Command = []string{"go"}
				client.Run.Args = []string{"run", "main.go"}

				pod, err := builder.PodForClient(client)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers).ToNot(BeEmpty())

				runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
				Expect(runContainer.Args).To(ContainElement(fmt.Sprintf("--driver_port=%d", config.DriverPort)))
			})
		})

		It("sets a pod anti-affinity", func() {
			// Note: this is a simple test to ensure the anti-affinity is set.
			// It does not confirm its properties are correct. This check is
			// meant to guard against accidental deletions of anti-affinities.
			pod, err := builder.PodForClient(client)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.Affinity).ToNot(BeNil())
			Expect(pod.Spec.Affinity.PodAntiAffinity).ToNot((BeNil()))
		})
	})

	Describe("PodForServer", func() {
		var server *grpcv1.Server

		BeforeEach(func() {
			server = &testSpec.Servers[0]
		})

		It("sets the namespace to match the test", func() {
			pod, err := builder.PodForServer(server)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Namespace).To(Equal(test.Namespace))
		})

		It("sets a label with the name of the load test", func() {
			pod, err := builder.PodForServer(server)
			Expect(err).ToNot(HaveOccurred())

			testName, ok := pod.ObjectMeta.Labels[config.LoadTestLabel]
			Expect(ok).To(BeTrue())
			Expect(testName).To(Equal(test.Name))
		})

		It("sets a label indicating it is a server", func() {
			pod, err := builder.PodForServer(server)
			Expect(err).ToNot(HaveOccurred())

			role, ok := pod.ObjectMeta.Labels[config.RoleLabel]
			Expect(ok).To(BeTrue())
			Expect(role).To(Equal(config.ServerRole))
		})

		It("sets a label with the name of the server", func() {
			pod, err := builder.PodForServer(server)
			Expect(err).ToNot(HaveOccurred())

			componentName, ok := pod.ObjectMeta.Labels[config.ComponentNameLabel]
			Expect(ok).To(BeTrue())
			Expect(componentName).To(Equal(*server.Name))
		})

		It("sets node selector to match pool", func() {
			server.Pool = optional.StringPtr("testing-pool")

			pod, err := builder.PodForServer(server)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.NodeSelector).ToNot(BeNil())
			Expect(pod.Spec.NodeSelector["pool"]).To(Equal(*server.Pool))
		})

		Context("clone init container", func() {
			It("contains an init container named clone when clone instructions are present", func() {
				server.Clone = new(grpcv1.Clone)
				server.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				server.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.CloneInitContainerName))
			})

			It("does not contain an init container named clone when clone instructions are not present", func() {
				server.Clone = nil

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).ToNot(ContainElement(config.CloneInitContainerName))
			})

			It("sets an environment variable with the git repository", func() {
				server.Clone = new(grpcv1.Clone)
				server.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				server.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)

				var gitRepoEnv *corev1.EnvVar
				for i := range cloneContainer.Env {
					env := &cloneContainer.Env[i]

					if env.Name == config.CloneRepoEnv {
						gitRepoEnv = env
					}
				}

				Expect(gitRepoEnv).ToNot(BeNil())
				Expect(gitRepoEnv.Value).To(Equal(*server.Clone.Repo))
			})

			It("sets an environment variable with the git ref", func() {
				server.Clone = new(grpcv1.Clone)
				server.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				server.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)

				var gitRefEnv *corev1.EnvVar
				for i := range cloneContainer.Env {
					env := &cloneContainer.Env[i]

					if env.Name == config.CloneGitRefEnv {
						gitRefEnv = env
					}
				}

				Expect(gitRefEnv).ToNot(BeNil())
				Expect(gitRefEnv.Value).To(Equal(*server.Clone.GitRef))
			})

			It("creates volume mount for workspace", func() {
				server.Clone = new(grpcv1.Clone)
				server.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				server.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)
				Expect(cloneContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(cloneContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})
		})

		Context("build init container", func() {
			It("contains a init container named build when build instructions are present", func() {
				server.Build = new(grpcv1.Build)
				server.Build.Command = []string{"go"}
				server.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.BuildInitContainerName))
			})

			It("does not contain an init container named build when build instructions are not present", func() {
				server.Build = nil

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).ToNot(ContainElement(config.BuildInitContainerName))
			})

			It("sets working directory to workspace", func() {
				server.Build = new(grpcv1.Build)
				server.Build.Command = []string{"go"}
				server.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.BuildInitContainerName))

				buildContainer := kubehelpers.ContainerForName(config.BuildInitContainerName, pod.Spec.InitContainers)
				Expect(buildContainer.WorkingDir).To(Equal(config.WorkspaceMountPath))
			})

			It("creates volume mount for workspace", func() {
				server.Build = new(grpcv1.Build)
				server.Build.Command = []string{"go"}
				server.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				buildContainer := kubehelpers.ContainerForName(config.BuildInitContainerName, pod.Spec.InitContainers)
				Expect(buildContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(buildContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})
		})

		Context("run container", func() {
			It("creates volume mount for workspace", func() {
				server.Run = grpcv1.Run{}
				server.Run.Command = []string{"go"}
				server.Run.Args = []string{"run", "main.go"}

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers).ToNot(BeEmpty())

				runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
				Expect(runContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(runContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})

			It("exposes the driver port", func() {
				server.Run = grpcv1.Run{}
				server.Run.Command = []string{"go"}
				server.Run.Args = []string{"run", "main.go"}

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers).ToNot(BeEmpty())

				runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
				Expect(getNames(runContainer.Ports)).To(ContainElement("driver"))
				Expect(getValue("driver", "ContainerPort", runContainer.Ports)).To(BeEquivalentTo(config.DriverPort))
			})

			It("appends the driver port command line argument", func() {
				server.Run = grpcv1.Run{}
				server.Run.Command = []string{"go"}
				server.Run.Args = []string{"run", "main.go"}

				pod, err := builder.PodForServer(server)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers).ToNot(BeEmpty())

				runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
				Expect(runContainer.Args).To(ContainElement(fmt.Sprintf("--driver_port=%d", config.DriverPort)))
			})
		})

		It("sets a pod anti-affinity", func() {
			// Note: this is a simple test to ensure the anti-affinity is set.
			// It does not confirm its properties are correct. This check is
			// meant to guard against accidental deletions of anti-affinities.
			pod, err := builder.PodForServer(server)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.Affinity).ToNot(BeNil())
			Expect(pod.Spec.Affinity.PodAntiAffinity).ToNot((BeNil()))
		})
	})

	Describe("PodForDriver", func() {
		var driver *grpcv1.Driver

		BeforeEach(func() {
			driver = testSpec.Driver
		})

		It("sets the namespace to match the test", func() {
			pod, err := builder.PodForDriver(driver)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Namespace).To(Equal(test.Namespace))
		})

		It("sets a label with the name of the load test", func() {
			pod, err := builder.PodForDriver(driver)
			Expect(err).ToNot(HaveOccurred())

			testName, ok := pod.ObjectMeta.Labels[config.LoadTestLabel]
			Expect(ok).To(BeTrue())
			Expect(testName).To(Equal(test.Name))
		})

		It("sets a label indicating it is a driver", func() {
			pod, err := builder.PodForDriver(driver)
			Expect(err).ToNot(HaveOccurred())

			role, ok := pod.ObjectMeta.Labels[config.RoleLabel]
			Expect(ok).To(BeTrue())
			Expect(role).To(Equal(config.DriverRole))
		})

		It("sets a label with the name of the driver", func() {
			pod, err := builder.PodForDriver(driver)
			Expect(err).ToNot(HaveOccurred())
			componentName, ok := pod.ObjectMeta.Labels[config.ComponentNameLabel]
			Expect(ok).To(BeTrue())
			Expect(componentName).To(Equal(*driver.Name))
		})

		It("sets node selector to match pool", func() {
			driver.Pool = optional.StringPtr("testing-pool")

			pod, err := builder.PodForDriver(driver)
			Expect(err).ToNot(HaveOccurred())

			Expect(pod.Spec.NodeSelector).ToNot(BeNil())
			Expect(pod.Spec.NodeSelector["pool"]).To(Equal(*driver.Pool))
		})

		Context("clone init container", func() {
			It("contains an init container named clone when clone instructions are present", func() {
				driver.Clone = new(grpcv1.Clone)
				driver.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				driver.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.CloneInitContainerName))
			})

			It("does not contain an init container named clone when clone instructions are not present", func() {
				driver.Clone = nil

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).ToNot(ContainElement(config.CloneInitContainerName))
			})

			It("sets an environment variable with the git repository", func() {
				driver.Clone = new(grpcv1.Clone)
				driver.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				driver.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)

				var gitRepoEnv *corev1.EnvVar
				for i := range cloneContainer.Env {
					env := &cloneContainer.Env[i]

					if env.Name == config.CloneRepoEnv {
						gitRepoEnv = env
					}
				}

				Expect(gitRepoEnv).ToNot(BeNil())
				Expect(gitRepoEnv.Value).To(Equal(*driver.Clone.Repo))
			})

			It("sets an environment variable with the git ref", func() {
				driver.Clone = new(grpcv1.Clone)
				driver.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				driver.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)

				var gitRefEnv *corev1.EnvVar
				for i := range cloneContainer.Env {
					env := &cloneContainer.Env[i]

					if env.Name == config.CloneGitRefEnv {
						gitRefEnv = env
					}
				}

				Expect(gitRefEnv).ToNot(BeNil())
				Expect(gitRefEnv.Value).To(Equal(*driver.Clone.GitRef))
			})

			It("creates volume mount for workspace", func() {
				driver.Clone = new(grpcv1.Clone)
				driver.Clone.Repo = optional.StringPtr("https://github.com/grpc/test-infra.git")
				driver.Clone.GitRef = optional.StringPtr("master")

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				cloneContainer := kubehelpers.ContainerForName(config.CloneInitContainerName, pod.Spec.InitContainers)
				Expect(cloneContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(cloneContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})
		})

		Context("build init container", func() {
			It("contains a init container named build when build instructions are present", func() {
				driver.Build = new(grpcv1.Build)
				driver.Build.Command = []string{"go"}
				driver.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.BuildInitContainerName))
			})

			It("does not contain an init container named build when build instructions are not present", func() {
				driver.Build = nil

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).ToNot(ContainElement(config.BuildInitContainerName))
			})

			It("sets working directory to workspace", func() {
				driver.Build = new(grpcv1.Build)
				driver.Build.Command = []string{"go"}
				driver.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())
				Expect(getNames(pod.Spec.InitContainers)).To(ContainElement(config.BuildInitContainerName))

				buildContainer := kubehelpers.ContainerForName(config.BuildInitContainerName, pod.Spec.InitContainers)
				Expect(buildContainer.WorkingDir).To(Equal(config.WorkspaceMountPath))
			})

			It("creates volume mount for workspace", func() {
				driver.Build = new(grpcv1.Build)
				driver.Build.Command = []string{"go"}
				driver.Build.Args = []string{"run", "main.go"}

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.InitContainers).ToNot(BeEmpty())

				buildContainer := kubehelpers.ContainerForName(config.BuildInitContainerName, pod.Spec.InitContainers)
				Expect(buildContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(buildContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})
		})

		Context("run container", func() {
			It("creates volume mount for workspace", func() {
				driver.Run = grpcv1.Run{}
				driver.Run.Command = []string{"go"}
				driver.Run.Args = []string{"run", "main.go"}

				pod, err := builder.PodForDriver(driver)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers).ToNot(BeEmpty())

				runContainer := kubehelpers.ContainerForName(config.RunContainerName, pod.Spec.Containers)
				Expect(runContainer.VolumeMounts).ToNot(BeEmpty())
				Expect(runContainer.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      config.WorkspaceVolumeName,
					MountPath: config.WorkspaceMountPath,
				}))
			})
		})

		It("sets a pod anti-affinity", func() {
			// Note: this is a simple test to ensure the anti-affinity is set.
			// It does not confirm its properties are correct. This check is
			// meant to guard against accidental deletions of anti-affinities.
			pod, err := builder.PodForDriver(driver)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.Affinity).ToNot(BeNil())
			Expect(pod.Spec.Affinity.PodAntiAffinity).ToNot((BeNil()))
		})
	})
})
