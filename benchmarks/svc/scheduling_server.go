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

	pb "github.com/grpc/test-infra/proto/grpc/testing"
	svcpb "github.com/grpc/test-infra/benchmarks/proto/scheduling/v1"
	"github.com/grpc/test-infra/benchmarks/svc/store"
	"github.com/grpc/test-infra/benchmarks/svc/types"

	lrpb "google.golang.org/genproto/googleapis/longrunning"
)

// Scheduler is a single method interface for queueing sessions.
type Scheduler interface {
	// Schedule enqueues a session, returning any immediate error.
	// Infrastructure and test runtime errors will not be
	// returned.
	Schedule(s *types.Session) error
}

// newSessionFunc is the type of the new session constructor.
type newSessionFunc func(c *types.Component, w []*types.Component, s *pb.Scenario) *types.Session

// SchedulingServer implements the scheduling service.
type SchedulingServer struct {
	scheduler  Scheduler
	operations lrpb.OperationsServer
	store      store.Store
	newSession newSessionFunc
}

// NewSchedulingServer constructs a scheduling server from a scheduler,
// an operations server, and a store.
func NewSchedulingServer(scheduler Scheduler, operations lrpb.OperationsServer, store store.Store) *SchedulingServer {
	return &SchedulingServer{
		scheduler:  scheduler,
		operations: operations,
		store:      store,
		newSession: types.NewSession,
	}
}

// StartTestSession implements the scheduling service interface for
// starting a test session.
func (s *SchedulingServer) StartTestSession(ctx context.Context, req *svcpb.StartTestSessionRequest) (operation *lrpb.Operation, err error) {
	driver := types.NewComponent(
		req.Driver.ContainerImage,
		types.DriverComponent,
	)
	driver.PoolName = req.Driver.Pool
	workers := make([]*types.Component, len(req.Workers))
	for i, v := range req.Workers {
		workers[i] = types.NewComponent(
			v.ContainerImage,
			types.ComponentKindFromProto(v.Kind),
		)
		workers[i].PoolName = v.Pool
	}
	session := s.newSession(driver, workers, req.Scenario)
	err = s.store.StoreSession(session)
	if err != nil {
		err = fmt.Errorf("error storing new test session: %v", err)
		return
	}
	// Latest event at session creation is nil.
	operation, err = newOperation(nil, session)
	if err != nil {
		err = fmt.Errorf(
			"Error creating operation for new test session: %v",
			err,
		)
		return
	}
	err = s.scheduler.Schedule(session)
	if err != nil {
		operation = nil
		s.store.DeleteSession(session.Name)
		err = fmt.Errorf(
			"error scheduling new test session: %v",
			err,
		)
		return
	}
	return
}
