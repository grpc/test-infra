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
	"fmt"

	"github.com/google/uuid"
	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// Defaults defines the default settings for the system.
type Defaults struct {
	// ComponentNamespace is the default namespace for load test components. Note
	// this is not the namespace for the manager.
	ComponentNamespace string `json:"componentNamespace"`

	// DefaultPoolLabels map a client, driver and server to a label on a node.
	// Any node with a matching key and a value of "true" may be used as a
	// default pool.
	DefaultPoolLabels *PoolLabelMap `json:"defaultPoolLabels,omitempty"`

	// CloneImage specifies the default container image to use for
	// cloning Git repositories at a specific snapshot.
	CloneImage string `json:"cloneImage"`

	// ReadyImage specifies the container image to use to block the driver from
	// starting before all worker pods are ready.
	ReadyImage string `json:"readyImage"`

	// DriverImage specifies a default driver image. This image will
	// be used to orchestrate a test.
	DriverImage string `json:"driverImage"`

	// Languages specifies the default build and run container images
	// for each known language.
	Languages []LanguageDefault `json:"languages,omitempty"`

	// KillAfterSeconds time allowed for pods to respond after timeout.
	KillAfterSeconds int64 `json:"killAfterSeconds"`
}

// Validate ensures that the required fields are present and an acceptable
// value. If an issue is encountered, an error is returned. If the defaults are
// valid, nil is returned.
func (d *Defaults) Validate() error {
	if d.CloneImage == "" {
		return errors.New("missing image for clone init container")
	}

	if d.ReadyImage == "" {
		return errors.New("missing image for ready init container")
	}

	if d.DriverImage == "" {
		return errors.New("missing image for driver container")
	}

	for i, ld := range d.Languages {
		if ld.Language == "" {
			return errors.Errorf("language (index %d) unnamed", i)
		}

		if ld.BuildImage == "" {
			return errors.Errorf("language %q (index %d) missing image for build init container", ld.Language, i)
		}

		if ld.RunImage == "" {
			return errors.Errorf("language %q (index %d) missing image for run container", ld.Language, i)
		}
	}

	if d.KillAfterSeconds == -1 {
		return errors.Errorf("killAfterSeconds missing")
	}

	return nil
}

// SetLoadTestDefaults applies default values for missing fields that are
// required to reconcile a load test.
//
// This returns an error if the system has no viable default. For example, the
// system cannot infer a run image for "fortran" if a build image was not
// declared for this language in the Defaults object.
func (d *Defaults) SetLoadTestDefaults(test *grpcv1.LoadTest) error {
	testSpec := &test.Spec
	im := newImageMap(d.Languages)

	if test.Namespace == "" {
		test.Namespace = d.ComponentNamespace
	}

	if err := d.setDriverDefaults(im, testSpec); err != nil {
		return errors.Wrap(err, "could not set defaults for driver")
	}

	for i := range testSpec.Servers {
		if err := d.setServerDefaults(im, &testSpec.Servers[i]); err != nil {
			return errors.Wrapf(err, "could not set defaults for server at index %d", i)
		}
	}

	for i := range testSpec.Clients {
		if err := d.setClientDefaults(im, &testSpec.Clients[i]); err != nil {
			return errors.Wrapf(err, "could not set defaults for client at index %d", i)
		}
	}

	return nil
}

// setCloneOrDefault sets the default clone image if it is unset.
func (d *Defaults) setCloneOrDefault(clone *grpcv1.Clone) {
	if clone != nil && clone.Image == nil {
		clone.Image = &d.CloneImage
	}
}

// setBuildOrDefault sets the default build image if it is unset. It returns an
// error if there is no default build image for the provided language.
func (d *Defaults) setBuildOrDefault(im *imageMap, language string, build *grpcv1.Build) error {
	if build != nil && build.Image == nil {
		buildImage, err := im.buildImage(language)
		if err != nil {
			return errors.Wrap(err, "could not infer default build image")
		}

		build.Image = &buildImage
	}

	return nil
}

// setRunOrDefault sets the default runtime image if it is unset. It returns an
// error if there is no default runtime image for the provided language.
func (d *Defaults) setRunOrDefault(im *imageMap, language string, run *grpcv1.Run) error {
	if run != nil && run.Image == nil {
		runImage, err := im.runImage(language)
		if err != nil {
			return errors.Wrap(err, "could not infer default run image")
		}

		run.Image = &runImage

		run.Env = append(run.Env, corev1.EnvVar{
			Name:  KillAfterSeconds,
			Value: fmt.Sprintf("%d", d.KillAfterSeconds),
		})
	}

	return nil
}

// setDriverDefaults sets default name, pool and container images for a driver.
// An error is returned if a default could not be inferred for a field.
func (d *Defaults) setDriverDefaults(im *imageMap, testSpec *grpcv1.LoadTestSpec) error {
	if testSpec.Driver == nil {
		testSpec.Driver = new(grpcv1.Driver)
	}

	driver := testSpec.Driver

	if driver.Language == "" {
		driver.Language = "cxx"
	}

	if driver.Run.Image == nil {
		driver.Run.Image = &d.DriverImage
	}

	driver.Name = unwrapStrOrUUID(driver.Name)
	d.setCloneOrDefault(driver.Clone)

	if err := d.setBuildOrDefault(im, driver.Language, driver.Build); err != nil {
		return errors.Wrap(err, "failed to set defaults on instructions to build the driver")
	}

	if err := d.setRunOrDefault(im, driver.Language, &driver.Run); err != nil {
		return errors.Wrap(err, "failed to set defaults on instructions to run the driver")
	}

	return nil
}

// setClientDefaults sets default name, pool and container images for a client.
// An error is returned if a default could not be inferred for a field.
func (d *Defaults) setClientDefaults(im *imageMap, client *grpcv1.Client) error {
	if client == nil {
		return errors.New("cannot set defaults on a nil client")
	}

	client.Name = unwrapStrOrUUID(client.Name)
	d.setCloneOrDefault(client.Clone)

	if err := d.setBuildOrDefault(im, client.Language, client.Build); err != nil {
		return errors.Wrap(err, "failed to set defaults on instructions to build the client")
	}

	if err := d.setRunOrDefault(im, client.Language, &client.Run); err != nil {
		return errors.Wrap(err, "failed to set defaults on instructions to run the client")
	}

	return nil
}

// setServersDefaults sets default name, pool and container images for a server.
// An error is returned if a default could not be inferred for a field.
func (d *Defaults) setServerDefaults(im *imageMap, server *grpcv1.Server) error {
	if server == nil {
		return errors.New("cannot set defaults on a nil server")
	}

	server.Name = unwrapStrOrUUID(server.Name)
	d.setCloneOrDefault(server.Clone)

	if err := d.setBuildOrDefault(im, server.Language, server.Build); err != nil {
		return errors.Wrap(err, "failed to set defaults on instructions to build the server")
	}

	if err := d.setRunOrDefault(im, server.Language, &server.Run); err != nil {
		return errors.Wrap(err, "failed to set defaults on instructions to run the server")
	}

	return nil
}

// unwrapStrOrUUID returns the string pointer if the pointer is not nil;
// otherwise, it returns a pointer to a UUID string. This method can be used to
// assign a unique name to a client, driver or server if one is not already set.
func unwrapStrOrUUID(namePtr *string) *string {
	if namePtr != nil {
		return namePtr
	}

	name := uuid.New().String()
	return &name
}

// LanguageDefault defines a programming language, as well as its
// default container images.
type LanguageDefault struct {
	// Language uniquely identifies a programming language. When the
	// system encounters this name, it will select the build image and
	// run image as the defaults.
	Language string `json:"language"`

	// BuildImage specifies the default container image for building or
	// assembling an executable or bundle for a language. This image
	// likely contains a compiler and any required libraries for
	// compilation.
	BuildImage string `json:"buildImage"`

	// RunImage specifies the default container image for the
	// environment for the runtime of the test. It should provide any
	// necessary interpreters or dependencies to run or use the output
	// of the build image.
	RunImage string `json:"runImage"`
}

// PoolLabelMap maps a client, driver or server to a string. This string should
// be the key of a label on a node where the client, driver or server pods may
// run. The value of the label should be the string "true".
//
// For example, the Driver field may be set to "default-driver-pool". This means
// the driver would run on any node with a "default-driver-pool" label set to
// a value of "true".
type PoolLabelMap struct {
	// Client maps a client to the key of a label where the client may run.
	Client string `json:"client"`

	// Driver maps a driver to the key of a label where the driver may run.
	Driver string `json:"driver"`

	// Server maps a server to the key of a label where the server may run.
	Server string `json:"server"`
}
