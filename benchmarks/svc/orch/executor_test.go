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
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1Fake "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

func TestKubeExecutorProvision(t *testing.T) {

	cases := []struct {
		description string
		events      []fakeWatchEvent
		ctxTimeout  time.Duration
		errors      bool
	}{
		{
			description: "successful pods",
			events: []fakeWatchEvent{
				{
					component: types.ServerComponent,
					health:    Ready,
					podIP:     "127.0.0.1",
				},
				{
					component: types.ClientComponent,
					health:    Ready,
					podIP:     "127.0.0.1",
				},
				{
					component: types.DriverComponent,
					health:    Ready,
					podIP:     "127.0.0.1",
				},
			},
			errors: false,
		},
		{
			description: "failed driver pod",
			events: []fakeWatchEvent{
				{
					component: types.ServerComponent,
					health:    Ready,
					podIP:     "127.0.0.1",
				},
				{
					component: types.ClientComponent,
					health:    Ready,
					podIP:     "127.0.0.1",
				},
				{
					component: types.DriverComponent,
					health:    Failed,
					podIP:     "127.0.0.1",
				},
			},
			errors: true,
		},
		{
			description: "cancelled context",
			ctxTimeout:  1 * time.Millisecond * timeMultiplier,
			events: []fakeWatchEvent{
				{
					component: types.ServerComponent,
					sleep:     10 * time.Second * timeMultiplier,
					health:    Succeeded,
					podIP:     "127.0.0.1",
				},
				{
					component: types.ClientComponent,
					sleep:     10 * time.Second * timeMultiplier,
					health:    Succeeded,
					podIP:     "127.0.0.1",
				},
				{
					component: types.DriverComponent,
					sleep:     10 * time.Second * timeMultiplier,
					health:    Succeeded,
					podIP:     "127.0.0.1",
				},
			},
			errors: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			fakePodInf := newFakePodInterface(t)

			server := types.NewComponent(testContainerImage, types.ServerComponent)
			client := types.NewComponent(testContainerImage, types.ClientComponent)
			driver := types.NewComponent(testContainerImage, types.DriverComponent)

			components := []*types.Component{server, client, driver}
			session := types.NewSession(driver, components[:2], nil)

			e := &kubeExecutor{
				name:    "provision-test-executor",
				pcd:     fakePodInf,
				watcher: nil,
				store:   nil,
			}
			eventChan := make(chan *PodWatchEvent)
			e.eventChan = eventChan
			e.session = session

			go func() {
				for _, event := range tc.events {
					var componentName string

					switch event.component {
					case types.ServerComponent:
						componentName = server.Name
					case types.ClientComponent:
						componentName = client.Name
					case types.DriverComponent:
						componentName = driver.Name
					default:
						componentName = "_UNKNOWN_"
					}

					time.Sleep(event.sleep)
					eventChan <- &PodWatchEvent{
						SessionName:   session.Name,
						ComponentName: componentName,
						Pod:           nil,
						PodIP:         event.podIP,
						Health:        event.health,
						Error:         nil,
					}
				}
			}()

			var ctx context.Context
			var cancel context.CancelFunc

			if tc.ctxTimeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tc.ctxTimeout)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			err := e.provision(ctx)
			if tc.errors {
				if err == nil {
					t.Fatalf("provision did not error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error in provision: %v", err)
			}

			pods, err := listPods(t, fakePodInf)
			if err != nil {
				t.Fatalf("could not list pods from provision: %v", err)
			}

			expectedNames := []string{driver.Name, server.Name, client.Name}
			for _, en := range expectedNames {
				found := false

				for _, pod := range pods {
					if strings.Compare(pod.ObjectMeta.Name, en) == 0 {
						found = true
					}
				}

				if !found {
					t.Errorf("provision did not create pod for component %v", en)
				}
			}
		})
	}
}

func TestKubeExecutorMonitor(t *testing.T) {
	cases := []struct {
		description  string
		event        *PodWatchEvent
		eventTimeout time.Duration
		ctxTimeout   time.Duration
		errors       bool
	}{
		{
			description: "success event received",
			event:       &PodWatchEvent{Health: Succeeded},
			errors:      false,
		},
		{
			description: "failure event received",
			event:       &PodWatchEvent{Health: Failed},
			errors:      true,
		},
		{
			description:  "cancelled context",
			event:        &PodWatchEvent{Health: Succeeded},
			eventTimeout: 10 * time.Second * timeMultiplier,
			ctxTimeout:   1 * time.Millisecond * timeMultiplier,
			errors:       true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			fakePodInf := newFakePodInterface(t)

			driver := types.NewComponent(testContainerImage, types.DriverComponent)
			server := types.NewComponent(testContainerImage, types.ServerComponent)
			client := types.NewComponent(testContainerImage, types.ClientComponent)

			components := []*types.Component{server, client, driver}
			session := types.NewSession(driver, components[:2], nil)

			e := &kubeExecutor{
				name:    "",
				pcd:     fakePodInf,
				watcher: nil,
				store:   nil,
			}
			eventChan := make(chan *PodWatchEvent)
			e.eventChan = eventChan
			e.session = session

			go func() {
				time.Sleep(tc.eventTimeout)
				eventChan <- &PodWatchEvent{
					SessionName:   session.Name,
					ComponentName: driver.Name,
					Pod:           tc.event.Pod,
					PodIP:         tc.event.PodIP,
					Health:        tc.event.Health,
					Error:         tc.event.Error,
				}
			}()

			var ctx context.Context
			var cancel context.CancelFunc

			if tc.ctxTimeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tc.ctxTimeout)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			err := e.monitor(ctx)
			if err == nil && tc.errors {
				t.Errorf("case '%v' did not return error", tc.description)
			} else if err != nil && !tc.errors {
				t.Errorf("case '%v' unexpectedly returned error '%v'", tc.description, err)
			}
		})
	}
}

type fakeWatchEvent struct {
	component types.ComponentKind
	health    Health
	sleep     time.Duration
	podIP     string
}

// TODO(@codeblooded): Refactor clean method, or choose to not test this method
//func TestKubeExecutorClean(t *testing.T) {
//	fakePodInf := newFakePodInterface(t)
//
//	driver := types.NewComponent(testContainerImage, types.DriverComponent)
//	server := types.NewComponent(testContainerImage, types.ServerComponent)
//	client := types.NewComponent(testContainerImage, types.ClientComponent)
//
//	session := types.NewSession(driver, []*types.Component{server, client}, nil)
//	driverPod := newSpecBuilder(session, driver).Pod()
//	serverPod := newSpecBuilder(session, server).Pod()
//	clientPod := newSpecBuilder(session, client).Pod()
//
//	fakePodInf.Create(driverPod)
//	fakePodInf.Create(serverPod)
//	fakePodInf.Create(clientPod)
//
//	pods, err := listPods(t, fakePodInf)
//	if err != nil {
//		t.Fatalf("could not list pods: %v", err)
//	}
//	podCountBefore := len(pods)
//
//	e := newKubeExecutor(0, fakePodInf, nil, nil)
//	e.session = session
//	if err = e.clean(fakePodInf); err != nil {
//		t.Fatalf("returned an error unexpectedly: %v", err)
//	}
//
//	pods, err = listPods(t, fakePodInf)
//	if err != nil {
//		t.Fatalf("could not list pods: %v", err)
//	}
//	podCountAfter := len(pods)
//
//	if podCountAfter-podCountBefore != 3 {
//		t.Fatalf("deletion did not remove all pods, %v remain", podCountAfter)
//	}
//}

func listPods(t *testing.T, fakePodInf *corev1Fake.FakePods) ([]corev1.Pod, error) {
	t.Helper()

	podList, err := fakePodInf.List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("setup failed, could not fetch pod list from kubernetes fake")
	}
	return podList.Items, nil
}
