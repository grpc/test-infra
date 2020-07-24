package defaults_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/pkg/defaults"
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
	var defaultOptions *defaults.Defaults

	BeforeEach(func() {
		loadtest = completeLoadTest.DeepCopy()

		defaultOptions = &defaults.Defaults{
			DriverPool:  "drivers",
			WorkerPool:  "workers-8core",
			DriverPort:  10000,
			ServerPort:  10010,
			CloneImage:  "gcr.io/grpc-fake-project/test-infra/clone",
			DriverImage: "gcr.io/grpc-fake-project/test-infra/driver",
			BuildImages: defaults.ImageMap{
				CXX:  "l.gcr.io/google/bazel:latest",
				Go:   "golang:1.14",
				Java: "gradle:jdk8",
			},
			RuntimeImages: defaults.ImageMap{
				CXX:  "gcr.io/grpc-fake-project/test-infra/cxx",
				Go:   "gcr.io/grpc-fake-project/test-infra/go",
				Java: "gcr.io/grpc-fake-project/test-infra/java",
			},
		}
	})

	Context("driver", func() {
		It("sets default when nil", func() {
			loadtest.Spec.Driver = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(copy.Spec.Driver).ToNot(BeNil())
			Expect(*copy.Spec.Driver.Run.Image).To(Equal(defaultOptions.DriverImage))
		})

		It("does not override when set", func() {
			driverImage := "gcr.io/grpc-example/test-image"
			loadtest.Spec.Driver.Run.Image = &driverImage
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Run.Image).To(Equal(driverImage))
		})

		It("sets default driver pool when nil", func() {
			loadtest.Spec.Driver.Pool = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Pool).To(Equal(defaultOptions.DriverPool))
		})

		It("does not override pool when set", func() {
			testPool := "preset-pool"
			loadtest.Spec.Driver.Pool = &testPool
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
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

			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Clone.Image).To(Equal(defaultOptions.CloneImage))
		})

		It("sets build image when missing", func() {
			build := new(grpcv1.Build)
			build.Image = nil
			build.Command = []string{"bazel"}

			loadtest.Spec.Driver.Build = build

			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Driver.Build.Image).To(Equal(defaultOptions.BuildImages.CXX))
		})

		It("errors when build image missing and language unknown", func() {
			build := new(grpcv1.Build)
			build.Image = nil

			loadtest.Spec.Driver.Language = "fortran"
			loadtest.Spec.Driver.Build = build

			_, err := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("servers", func() {
		It("sets clone image when missing", func() {
			loadtest.Spec.Servers[0].Clone.Image = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Servers[0].Clone.Image).To(Equal(defaultOptions.CloneImage))
		})

		It("sets build image when missing", func() {
			loadtest.Spec.Servers[0].Build.Image = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Servers[0].Build.Image).To(Equal(defaultOptions.BuildImages.CXX))
		})

		It("does not override pool when set", func() {
			pool := "custom-pool"
			loadtest.Spec.Servers[0].Pool = &pool
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Servers[0].Pool).To(Equal(pool))
		})

		It("sets default worker pool when nil", func() {
			loadtest.Spec.Servers[0].Pool = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Servers[0].Pool).To(Equal(defaultOptions.WorkerPool))
		})

		It("errors when run image missing and language unknown", func() {
			loadtest.Spec.Servers[0].Language = "fortran"
			loadtest.Spec.Servers[0].Run.Image = nil
			_, err := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})

		It("errors when build image missing and language unknown", func() {
			loadtest.Spec.Servers[0].Language = "fortran"
			loadtest.Spec.Servers[0].Build.Image = nil
			_, err := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("clients", func() {
		It("sets clone image when missing", func() {
			loadtest.Spec.Clients[0].Clone.Image = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Clients[0].Clone.Image).To(Equal(defaultOptions.CloneImage))
		})

		It("sets build image when missing", func() {
			loadtest.Spec.Clients[0].Build.Image = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Clients[0].Build.Image).To(Equal(defaultOptions.BuildImages.CXX))
		})

		It("does not override pool when set", func() {
			pool := "custom-pool"
			loadtest.Spec.Clients[0].Pool = &pool
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Clients[0].Pool).To(Equal(pool))
		})

		It("sets default worker pool when nil", func() {
			loadtest.Spec.Clients[0].Pool = nil
			copy, _ := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(*copy.Spec.Clients[0].Pool).To(Equal(defaultOptions.WorkerPool))
		})

		It("errors when run image missing and language unknown", func() {
			loadtest.Spec.Clients[0].Language = "fortran"
			loadtest.Spec.Clients[0].Run.Image = nil
			_, err := defaults.CopyWithDefaults(defaultOptions, loadtest)
			Expect(err).To(HaveOccurred())
		})

		It("errors when build image missing and language unknown", func() {
			loadtest.Spec.Clients[0].Language = "fortran"
			loadtest.Spec.Clients[0].Build.Image = nil
			_, err := defaults.CopyWithDefaults(defaultOptions, loadtest)
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

	return &grpcv1.LoadTest{
		Spec: grpcv1.LoadTestSpec{
			Driver: &grpcv1.Driver{
				Component: grpcv1.Component{
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
