package defaults

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// CopyWithDefaults makes a deep copy of a load test, applying default values
// for fields.
//
// This returns an error if the system has no viable default. For example, the
// system cannot infer a run image for "fortran" if a build image was not
// declared for this language in the Defaults object.
func CopyWithDefaults(d *Defaults, loadtest *grpcv1.LoadTest) (*grpcv1.LoadTest, error) {
	im := newImageMap(d.Languages)
	test := loadtest.DeepCopy()
	spec := &test.Spec

	if test.Namespace == "" {
		test.Namespace = corev1.NamespaceDefault
	}

	if spec.Driver == nil {
		spec.Driver = &grpcv1.Driver{
			Component: grpcv1.Component{
				Language: "cxx",
				Run: grpcv1.Run{
					Image: &d.DriverImage,
				},
			},
		}
	}

	if spec.Driver.Name == nil {
		setRandomName(&spec.Driver.Component)
	}

	if spec.Driver.Pool == nil {
		spec.Driver.Pool = &d.DriverPool
	}

	if spec.Driver.Clone != nil {
		if spec.Driver.Clone.Image == nil {
			spec.Driver.Clone.Image = &d.CloneImage
		}
	}

	if spec.Driver.Build != nil {
		if spec.Driver.Build.Image == nil {
			language := spec.Driver.Language

			image, err := im.buildImage(language)
			if err != nil {
				return nil, errors.Wrap(err, "could not set default driver build image for unknown language")
			}

			spec.Driver.Build.Image = &image
		}
	}

	var servers []grpcv1.Server
	for _, server := range spec.Servers {
		if server.Name == nil {
			setRandomName(&server.Component)
		}

		if server.Pool == nil {
			server.Pool = &d.WorkerPool
		}

		if server.Run.Image == nil {
			language := server.Language

			image, err := im.runImage(language)
			if err != nil {
				return nil, errors.Wrap(err, "could not set default runtime image for server in unknown language")
			}

			server.Run.Image = &image
		}

		if server.Clone != nil {
			if server.Clone.Image == nil {
				server.Clone.Image = &d.CloneImage
			}
		}

		if server.Build != nil {
			if server.Build.Image == nil {
				language := server.Language

				image, err := im.buildImage(language)
				if err != nil {
					return nil, errors.Wrap(err, "could not set default server build image for unknown language")
				}

				server.Build.Image = &image
			}
		}

		servers = append(servers, server)
	}
	spec.Servers = servers

	var clients []grpcv1.Client
	for _, client := range spec.Clients {
		if client.Name == nil {
			setRandomName(&client.Component)
		}

		if client.Pool == nil {
			client.Pool = &d.WorkerPool
		}

		if client.Run.Image == nil {
			language := client.Language

			image, err := im.runImage(language)
			if err != nil {
				return nil, errors.Wrap(err, "could not set default runtime image for client in unknown language")
			}

			client.Run.Image = &image
		}

		if client.Clone != nil {
			if client.Clone.Image == nil {
				client.Clone.Image = &d.CloneImage
			}
		}

		if client.Build != nil {
			if client.Build.Image == nil {
				language := client.Language

				image, err := im.buildImage(language)
				if err != nil {
					return nil, errors.Wrap(err, "could not set default client build image for unknown language")
				}

				client.Build.Image = &image
			}
		}

		clients = append(clients, client)
	}
	spec.Clients = clients

	return test, nil
}

// setRandomName generates a new globally unique name and sets it on a
// component.
//
// Note this function will panic if given a nil pointer. The caller should
// ensure it is not passed nil arguments. This is meant to be an internal helper
// to CopyWithDefaults, and it is likely to be inlined by the Go compiler.
func setRandomName(component *grpcv1.Component) {
	name := uuid.New().String()
	component.Name = &name
}
