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

package svc

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/grpc/test-infra/proto/grpc/testing"
	svcpb "github.com/grpc/test-infra/benchmarks/proto/scheduling/v1"
	"github.com/grpc/test-infra/benchmarks/svc/store"
	"github.com/grpc/test-infra/benchmarks/svc/types"
)

type schedulingTestExpected struct {
	ReturnsError bool
}

type schedulingTestInfo struct {
	TestName        string
	SessionExists   bool
	SchedulingError error
	Expected        *schedulingTestExpected
}

type FakeScheduler struct {
	Sessions []*types.Session
	Err      error
}

func newFakeScheduler() *FakeScheduler {
	return &FakeScheduler{
		Sessions: make([]*types.Session, 0),
	}
}

func (f *FakeScheduler) Schedule(s *types.Session) error {
	f.Sessions = append(f.Sessions, s)
	return f.Err
}

func newFakeNewSessionFunc(sessionName string) newSessionFunc {
	f := func(c *types.Component, w []*types.Component, s *pb.Scenario) *types.Session {
		session := types.NewSession(c, w, s)
		session.Name = sessionName
		return session
	}
	return f
}

func newSchedulingTestInfos() []*schedulingTestInfo {
	return []*schedulingTestInfo{
		{
			TestName: "Success",
			Expected: &schedulingTestExpected{},
		},
		{
			TestName:      "StorageError",
			SessionExists: true,
			Expected: &schedulingTestExpected{
				ReturnsError: true,
			},
		},
		{
			TestName:        "SchedulingError",
			SchedulingError: fmt.Errorf("scheduling error"),
			Expected: &schedulingTestExpected{
				ReturnsError: true,
			},
		},
	}
}

func TestSchedulingServer(t *testing.T) {
	tis := newSchedulingTestInfos()
	for _, v := range tis {
		ti := v
		t.Run(ti.TestName, func(t *testing.T) {
			t.Parallel()
			storageServer := store.NewStorageServer()
			if ti.SessionExists {
				dup := newTestSession(1)
				dup.Name = "session-0"
				storageServer.StoreSession(dup)
			}
			operationsServer := NewOperationsServer(storageServer)
			fakeScheduler := newFakeScheduler()
			if ti.SchedulingError != nil {
				fakeScheduler.Err = ti.SchedulingError
			}
			ts := &SchedulingServer{
				scheduler:  fakeScheduler,
				operations: operationsServer,
				store:      storageServer,
				newSession: newFakeNewSessionFunc("session-0"),
			}
			ctx := context.TODO()
			session := newTestSession(0)
			req := &svcpb.StartTestSessionRequest{
				Scenario: session.Scenario,
			}
			req.Driver = &svcpb.Component{
				ContainerImage: session.Driver.ContainerImage,
				Kind:           session.Driver.Kind.Proto(),
			}
			req.Workers = make([]*svcpb.Component, len(session.Workers))
			for i, w := range session.Workers {
				req.Workers[i] = &svcpb.Component{
					ContainerImage: w.ContainerImage,
					Kind:           w.Kind.Proto(),
				}
			}
			op, err := ts.StartTestSession(ctx, req)
			if (err != nil) && !ti.Expected.ReturnsError {
				t.Fatalf("unexpected error starting session: %s", err)
			}
			if ti.SessionExists && (len(fakeScheduler.Sessions) > 0) {
				t.Fatalf("scheduler called for session that already exists: %s", err)
			}
			if (ti.SchedulingError != nil) && (len(fakeScheduler.Sessions) != 1) {
				t.Fatalf("expected scheduler to be called once, called %d times",
					len(fakeScheduler.Sessions))
			}
			if (err != nil) && ti.Expected.ReturnsError {
				return
			}
			if (err == nil) && (op == nil) {
				t.Fatalf("expected operation or error, received nil, nil")
			}
			operationName := "operations/session-0"
			if op.Name != operationName {
				t.Fatalf("expected operation name %q, received %q",
					operationName, op.Name)
			}
		})
	}
}
