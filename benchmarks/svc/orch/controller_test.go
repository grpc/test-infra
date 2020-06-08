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
	"testing"
	"time"

	"github.com/grpc/test-infra/benchmarks/svc/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewController(t *testing.T) {
	t.Run("nil clientset returns error", func(t *testing.T) {
		controller, err := NewController(nil, nil, nil)
		if err == nil {
			t.Errorf("no error returned for nil clientset")
		}
		if controller != nil {
			t.Errorf("controller instance returned despite nil clientset")
		}
	})
}

func TestControllerSchedule(t *testing.T) {
	cases := []struct {
		description string
		session     *types.Session
		start       bool
		shouldError bool
	}{
		{
			description: "session nil",
			session:     nil,
			start:       true,
			shouldError: true,
		},
		{
			description: "without controller start",
			session:     makeSessions(t, 1)[0],
			start:       false,
			shouldError: true,
		},
		{
			description: "valid session and controller start",
			session:     makeSessions(t, 1)[0],
			start:       true,
			shouldError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			controller, _ := NewController(fake.NewSimpleClientset(), nil, nil)
			executor := &executorMock{}
			controller.newExecutorFunc = func() Executor {
				return executor
			}

			if tc.start {
				controller.Start()
				defer controller.Stop(context.Background())
			}

			err := controller.Schedule(tc.session)
			if tc.shouldError && err == nil {
				t.Errorf("did not return an expected error")
			} else if !tc.shouldError {
				if err != nil {
					t.Fatalf("unexpected error returned: %v", err)
				}

				time.Sleep(100 * time.Millisecond * timeMultiplier)
				got := executor.session()
				if got == nil {
					t.Fatalf("expected executor to receive session %v, but it did not", tc.session.Name)
				}
				if got.Name != tc.session.Name {
					t.Fatalf("expected executor to receive session %v, but got %v", tc.session.Name, got.Name)
				}
			}
		})
	}
}

func TestControllerStart(t *testing.T) {
	t.Run("sets running state", func(t *testing.T) {
		controller, _ := NewController(fake.NewSimpleClientset(), nil, nil)
		controller.Start()
		defer controller.Stop(context.Background())
		if controller.Stopped() {
			t.Errorf("Stopped unexpectedly returned true after starting controller")
		}
	})

	cases := []struct {
		description string
		mockNL      *nodeListerMock
		mockPW      *podWatcherMock
		shouldError bool
	}{
		{
			description: "setup queue error",
			mockNL:      &nodeListerMock{err: errors.New("fake kubernetes error")},
			shouldError: true,
		},
		{
			description: "setup watcher error",
			mockPW:      &podWatcherMock{err: errors.New("fake kubernetes error")},
			shouldError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			controller, _ := NewController(fake.NewSimpleClientset(), nil, nil)
			controller.waitQueue = newQueue(limitlessTracker{})

			if tc.mockNL != nil {
				controller.nl = tc.mockNL
			}

			if tc.mockPW != nil {
				controller.pw = tc.mockPW
				controller.watcher = NewWatcher(tc.mockPW, nil)
			}

			err := controller.Start()
			if tc.shouldError && err == nil {
				t.Errorf("did not return an expected error")
			} else if !tc.shouldError && err != nil {
				t.Errorf("unexpected error returned: %v", err)
			}
		})
	}
}

func TestControllerStop(t *testing.T) {
	timeout := 100 * time.Millisecond * timeMultiplier
	fastTimeout := timeout / 3
	bottleneckTimeout := timeout * 3

	cases := []struct {
		description          string
		runningExecutorCount int
		executorTimeout      time.Duration
		stopTimeout          time.Duration
		shouldError          bool
	}{
		{
			description:          "no executors",
			runningExecutorCount: 0,
			stopTimeout:          bottleneckTimeout,
			shouldError:          false,
		},
		{
			description:          "one executor finishes",
			runningExecutorCount: 1,
			executorTimeout:      fastTimeout,
			stopTimeout:          bottleneckTimeout,
			shouldError:          false,
		},
		{
			description:          "one executor exheeds timeout",
			runningExecutorCount: 1,
			executorTimeout:      bottleneckTimeout,
			stopTimeout:          fastTimeout,
			shouldError:          true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			controller, _ := NewController(fake.NewSimpleClientset(), nil, nil)
			controller.running = true
			controller.waitQueue = newQueue(limitlessTracker{})

			var executors []Executor
			for i := 0; i < tc.runningExecutorCount; i++ {
				executors = append(executors, &executorMock{
					sideEffect: func() {
						time.Sleep(tc.executorTimeout * timeMultiplier)
					},
				})
			}

			index := -1
			controller.newExecutorFunc = func() Executor {
				index++
				return executors[index]
			}

			sessions := makeSessions(t, tc.runningExecutorCount)
			for _, session := range sessions {
				controller.Schedule(session)
			}

			go controller.loop()
			time.Sleep(timeout)

			ctx, cancel := context.WithTimeout(context.Background(), tc.stopTimeout)
			defer cancel()

			err := controller.Stop(ctx)
			if tc.shouldError && err == nil {
				t.Errorf("executors unexpectedly finished before timeout")
			} else if !tc.shouldError && err != nil {
				t.Errorf("timeout unexpectedly reached before executors done signal")
			}

			// try to schedule session after stopping
			session := makeSessions(t, 1)[0]
			if err = controller.Schedule(session); err == nil {
				t.Errorf("scheduling a session did not return an error after stop invoked")
			}
		})
	}
}
