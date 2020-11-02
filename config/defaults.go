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
	"github.com/google/uuid"
	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/pkg/errors"
)

// Defaults defines the default settings for the system.
type Defaults struct {
	// ComponentNamespace is the default namespace for load test components. Note
	// this is not the namespace for the manager.
	ComponentNamespace string `json:"componentNamespace"`

	// DriverPool is the name of a pool where driver components should
	// be scheduled by default.
	DriverPool string `json:"driverPool"`

	// WorkerPool is the name of a pool where server and client
	// components should be scheduled by default.
	WorkerPool string `json:"workerPool"`

	// DriverPort is the port through which the driver and workers
	// communicate.
	DriverPort int32 `json:"driverPort"`

	// ServerPort is the port through which a server component accepts
	// traffic from a client component.
	ServerPort int32 `json:"serverPort"`

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
}

// Validate ensures that the required fields are present and an acceptable
// value. If an issue is encountered, an error is returned. If the defaults are
// valid, nil is returned.
func (d *Defaults) Validate() error {
	var tcpPortMax int32 = 65535

	if d.DriverPool == "" {
		return errors.New("missing driver pool")
	}

	if d.WorkerPool == "" {
		return errors.New("missing worker pool")
	}

	if d.DriverPort < 0 || d.DriverPort > tcpPortMax {
		return errors.Errorf("driver port is outside of TCP range: [0, %d]", tcpPortMax)
	}

	if d.ServerPort < 0 || d.ServerPort > tcpPortMax {
		return errors.Errorf("server port is outside of TCP range: [0, %d]", tcpPortMax)
	}

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

	return nil
}

// SetLoadTestDefaults applies default values for missing fields that are
// required to reconcile a load test.
//
// This returns an error if the system has no viable default. For example, the
// system cannot infer a run image for "fortran" if a build image was not
// declared for this language in the Defaults object.
func (d *Defaults) SetLoadTestDefaults(test *grpcv1.LoadTest) error {
	var err error

	spec := &test.Spec

	if test.Namespace == "" {
		test.Namespace = d.ComponentNamespace
	}

	if spec.Driver == nil {
		spec.Driver = new(grpcv1.Driver)
	}

	if spec.Driver.Language == "" {
		// TODO: Make default driver language a configuration option
		spec.Driver.Language = "cxx"
	}

	if spec.Driver.Run.Image == nil {
		spec.Driver.Run.Image = &d.DriverImage
	}
	if err = d.setComponentDefaults(&spec.Driver.Component, d.DriverPool); err != nil {
		return errors.Wrap(err, "could not set defaults for driver")
	}

	for i := range spec.Servers {
		if err = d.setComponentDefaults(&spec.Servers[i].Component, d.WorkerPool); err != nil {
			return errors.Wrapf(err, "could not set defaults for server at index %d", i)
		}
	}

	for i := range spec.Clients {
		if err = d.setComponentDefaults(&spec.Clients[i].Component, d.WorkerPool); err != nil {
			return errors.Wrapf(err, "could not set defaults for client at index %d", i)
		}
	}

	return nil
}

// setComponentDefaults sets default name, pool and container images for a
// component. An error is returned if a default could not be inferred for a
// field.
func (d *Defaults) setComponentDefaults(component *grpcv1.Component, defaultPool string) error {
	language := component.Language
	im := newImageMap(d.Languages)

	if component.Name == nil {
		name := uuid.New().String()
		component.Name = &name
	}

	if component.Pool == nil {
		component.Pool = &defaultPool
	}

	if component.Clone != nil && component.Clone.Image == nil {
		component.Clone.Image = &d.CloneImage
	}

	build := component.Build
	if build != nil && build.Image == nil {
		buildImage, err := im.buildImage(language)
		if err != nil {
			return errors.Wrap(err, "could not infer default build image")
		}

		build.Image = &buildImage
	}

	run := &component.Run
	if run.Image == nil {
		runImage, err := im.runImage(language)
		if err != nil {
			return errors.Wrap(err, "could not infer default run image")
		}

		run.Image = &runImage
	}

	return nil
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
