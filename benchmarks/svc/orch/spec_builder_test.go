// Copyright 2020 gRPC authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orch

import (
	"reflect"
	"testing"

	apiv1 "k8s.io/api/core/v1"

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

func TestSpecBuilderContainers(t *testing.T) {
	image := "debian:jessie"
	component := types.NewComponent(image, types.DriverComponent)
	session := types.NewSession(component, nil, nil)
	sb := newSpecBuilder(session, component)
	containers := sb.Containers()

	if len(containers) < 1 {
		t.Fatalf("specBuilder Containers did not specify any containers; expected '%s'", image)
	}

	if actualImage := containers[0].Image; actualImage != image {
		t.Errorf("specBuilder Containers did not correctly set the container image; expected '%s' but got '%s'", image, actualImage)
	}
}

func TestSpecBuilderContainerPorts(t *testing.T) {
	cases := []struct {
		kind  types.ComponentKind
		ports []int32
	}{
		{types.DriverComponent, []int32{driverPort}},
		{types.ClientComponent, []int32{driverPort}},
		{types.ServerComponent, []int32{driverPort, serverPort}},
	}

	var containerPortSlice = func(cps []apiv1.ContainerPort) []int32 {
		var ports []int32

		for _, port := range cps {
			ports = append(ports, port.ContainerPort)
		}

		return ports
	}

	for _, c := range cases {
		component := types.NewComponent(testContainerImage, c.kind)
		var session *types.Session
		if component.Kind == types.DriverComponent {
			session = types.NewSession(component, nil, nil)
		} else {
			session = types.NewSession(nil, []*types.Component{component}, nil)
		}
		sb := newSpecBuilder(session, component)
		ports := containerPortSlice(sb.ContainerPorts())

		if !reflect.DeepEqual(ports, c.ports) {
			t.Errorf("specBuilder ContainerPorts does not contain the correct ports for %s; expected %v but got %v", c.kind, c.ports, ports)
		}
	}
}

func TestSpecBuilderLabels(t *testing.T) {
	// Check that spec contains a 'session-name' label
	component := types.NewComponent(testContainerImage, types.DriverComponent)
	session := types.NewSession(component, nil, nil)
	sb := newSpecBuilder(session, component)
	labels := sb.Labels()

	if sessionName := labels["session-name"]; sessionName != session.Name {
		t.Errorf("specBuilder Labels generated incorrect 'session-name' label; expected '%s' but got '%v'", session.Name, sessionName)
	}

	// Check the spec contains 'component-name' label
	component = types.NewComponent(testContainerImage, types.DriverComponent)
	session = types.NewSession(component, nil, nil)
	sb = newSpecBuilder(session, component)
	labels = sb.Labels()

	if componentName := labels["component-name"]; componentName != component.Name {
		t.Errorf("specBuilder Labels generated incorrect 'component-name' label; expected '%s' but got '%v'", component.Name, componentName)
	}

	// Check the spec constains 'component-kind' label
	kindCases := []struct {
		kind       types.ComponentKind
		labelValue string
	}{
		{types.DriverComponent, "driver"},
		{types.ClientComponent, "client"},
		{types.ServerComponent, "server"},
	}

	for _, c := range kindCases {
		component := types.NewComponent(testContainerImage, c.kind)
		var session *types.Session
		if c.kind == types.DriverComponent {
			session = types.NewSession(component, nil, nil)
		} else {
			session = types.NewSession(nil, []*types.Component{component}, nil)
		}
		sb := newSpecBuilder(session, component)
		labels := sb.Labels()

		if kind := labels["component-kind"]; kind != c.labelValue {
			t.Errorf("specBuilder Labels generated incorrect 'component-kind' label for %s component; expected '%s' but got '%v'", c.kind.String(), c.labelValue, kind)
		}
	}

	// Check that the 'autogen' label exists, signifying that this resource was automatically generated
	component = types.NewComponent(testContainerImage, types.DriverComponent)
	session = types.NewSession(component, nil, nil)
	sb = newSpecBuilder(session, component)
	labels = sb.Labels()

	if autogen := labels["autogen"]; autogen != "1" {
		t.Errorf("specBuilder Labels missing 'autogen' label to signify generated component")
	}
}

func TestSpecBuilderObjectMeta(t *testing.T) {
	component := types.NewComponent(testContainerImage, types.DriverComponent)
	componentName := component.Name
	session := types.NewSession(component, nil, nil)
	sb := newSpecBuilder(session, component)

	if resourceName := sb.ObjectMeta().Name; resourceName != componentName {
		t.Errorf("specBuilder ObjectMeta did not set the K8s resource name to the component name; expected '%s' but got '%s'", componentName, resourceName)
	}
}

func TestSpecBuilderEnv(t *testing.T) {
	// check all component env variables are copied to spec
	key := "TESTING"
	value := "true"
	component := types.NewComponent(testContainerImage, types.DriverComponent)
	component.Env = make(map[string]string)
	component.Env[key] = value
	session := types.NewSession(component, nil, nil)

	sb := newSpecBuilder(session, component)
	got := getEnv(sb.Env(), key)
	if got == nil || *got != value {
		t.Errorf("specBuilder Env did not copy all component env variables")
	}

	// check SCENARIO_JSON is always and only set on driver
	scenarioCases := []struct {
		componentKind   types.ComponentKind
		includeScenario bool
	}{
		{types.DriverComponent, true},
		{types.ServerComponent, false},
		{types.ClientComponent, false},
	}

	for _, c := range scenarioCases {
		component := types.NewComponent(testContainerImage, c.componentKind)
		var session *types.Session
		if component.Kind == types.DriverComponent {
			session = types.NewSession(component, nil, nil)
		} else {
			session = types.NewSession(nil, []*types.Component{component}, nil)
		}
		sb := newSpecBuilder(session, component)
		included := getEnv(sb.Env(), "SCENARIO_JSON") != nil

		if included != c.includeScenario {
			if c.includeScenario {
				t.Errorf("specBuilder Env did not set $SCENARIO_JSON env variable for %v", c.componentKind)
			} else {
				t.Errorf("specBuilder Env unexpectedly set $SCENARIO_JSON env variable for %v", c.componentKind)
			}
		}
	}

	// check WORKER_KIND is properly set on server and client
	clientValue := "client"
	serverValue := "server"
	kindCases := []struct {
		componentKind types.ComponentKind
		workerKind    *string
	}{
		{types.DriverComponent, nil},
		{types.ClientComponent, &clientValue},
		{types.ServerComponent, &serverValue},
	}

	for _, c := range kindCases {
		component := types.NewComponent(testContainerImage, c.componentKind)
		var session *types.Session
		if component.Kind == types.DriverComponent {
			session = types.NewSession(component, nil, nil)
		} else {
			session = types.NewSession(nil, []*types.Component{component}, nil)
		}
		sb := newSpecBuilder(session, component)
		got := getEnv(sb.Env(), "WORKER_KIND")

		if !reflect.DeepEqual(got, c.workerKind) {
			t.Errorf("expected WORKER_KIND to be '%v' for %v component, but got '%v'", strUnwrap(c.workerKind), c.componentKind, strUnwrap(got))
		}
	}
}

func getEnv(envs []apiv1.EnvVar, name string) *string {
	for _, env := range envs {
		if env.Name == name {
			return &env.Value
		}
	}
	return nil
}
