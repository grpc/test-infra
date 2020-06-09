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
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	pb "github.com/grpc/test-infra/proto/grpc/testing"
	"github.com/golang/protobuf/ptypes"
	svcpb "github.com/grpc/test-infra/benchmarks/proto/scheduling/v1"
	"github.com/grpc/test-infra/benchmarks/svc/store"
	"github.com/grpc/test-infra/benchmarks/svc/types"
	lrpb "google.golang.org/genproto/googleapis/longrunning"
	codepb "google.golang.org/genproto/googleapis/rpc/code"
)

type opsTestExpected struct {
	ReturnsError bool
	Done         bool
	HasResponse  bool
	StatusCode   codepb.Code
}

type opsTestInfo struct {
	TestName      string
	OperationName string
	Session       *types.Session
	Event         *types.Event
	Expected      *opsTestExpected
}

func newTestComponent(componentKind types.ComponentKind, sessionIndex int32) *types.Component {
	componentBaseName := strings.ToLower(componentKind.String())
	componentName := fmt.Sprintf("%s-%d",
		componentBaseName, sessionIndex)
	imageName := fmt.Sprintf("%s-image",
		componentName)
	return &types.Component{
		Name:           componentName,
		ContainerImage: imageName,
		Kind:           componentKind,
	}
}

func newTestScenario(sessionIndex int32) *pb.Scenario {
	return &pb.Scenario{
		Name: fmt.Sprintf("scenario-%d", sessionIndex),
	}
}

func newTestSession(index int32) *types.Session {
	sessionName := fmt.Sprintf("session-%d", index)
	driver := newTestComponent(types.DriverComponent, index)
	workers := make([]*types.Component, 2)
	workers[0] = newTestComponent(types.ClientComponent, index)
	workers[1] = newTestComponent(types.ServerComponent, index)
	scenario := newTestScenario(index)
	return &types.Session{
		Name:     sessionName,
		Driver:   driver,
		Workers:  workers,
		Scenario: scenario,
	}
}

func newTestLogBytes(sessionIndex int32, eventKind types.EventKind) []byte {
	logBytes := []byte(fmt.Sprintf("Test logs for Session %d", sessionIndex))
	switch eventKind {
	case types.InternalErrorEvent:
		return logBytes
	case types.DoneEvent:
		return logBytes
	case types.ErrorEvent:
		return logBytes
	default:
		return nil
	}
}

func newTestEvent(sessionIndex int32, kind types.EventKind) *types.Event {
	subjectName := fmt.Sprintf("event-%d", sessionIndex)
	description := fmt.Sprintf("Test event for Session %d", sessionIndex)
	driverLogs := newTestLogBytes(sessionIndex, kind)
	return &types.Event{
		SubjectName: subjectName,
		Kind:        kind,
		Description: description,
		DriverLogs:  driverLogs,
	}
}

func newOpsTestInfoInvalidOperation(expected *opsTestExpected) *opsTestInfo {
	return &opsTestInfo{
		TestName:      "InvalidOperation",
		OperationName: "invalid-operation-name",
		Expected:      expected,
	}
}

func newOpsTestInfoNoSession(index int32, expected *opsTestExpected) *opsTestInfo {
	session := newTestSession(index)
	operationName := getOperationName(session.Name)
	return &opsTestInfo{
		TestName:      "NoSession",
		OperationName: operationName,
		Expected:      expected,
	}
}

func newOpsTestInfoNoEvent(index int32, expected *opsTestExpected) *opsTestInfo {
	session := newTestSession(index)
	operationName := getOperationName(session.Name)
	return &opsTestInfo{
		TestName:      "SessionNoEvent",
		OperationName: operationName,
		Session:       session,
		Expected:      expected,
	}
}

func newOpsTestInfo(index int32, eventKind types.EventKind, expected *opsTestExpected) *opsTestInfo {
	session := newTestSession(index)
	event := newTestEvent(index, eventKind)
	testName := fmt.Sprintf("EventKind=%s", eventKind)
	operationName := getOperationName(session.Name)
	return &opsTestInfo{
		TestName:      testName,
		OperationName: operationName,
		Session:       session,
		Event:         event,
		Expected:      expected,
	}
}

func newOpsTestInfos() []*opsTestInfo {
	return []*opsTestInfo{
		newOpsTestInfoInvalidOperation(&opsTestExpected{
			ReturnsError: true,
		}),
		newOpsTestInfoNoSession(0, &opsTestExpected{
			ReturnsError: true,
		}),
		newOpsTestInfoNoEvent(1, &opsTestExpected{}),
		newOpsTestInfo(2, types.InternalErrorEvent, &opsTestExpected{
			Done:       true,
			StatusCode: codepb.Code_INTERNAL,
		}),
		newOpsTestInfo(3, types.QueueEvent, &opsTestExpected{}),
		newOpsTestInfo(4, types.AcceptEvent, &opsTestExpected{}),
		newOpsTestInfo(5, types.ProvisionEvent, &opsTestExpected{}),
		newOpsTestInfo(6, types.RunEvent, &opsTestExpected{}),
		newOpsTestInfo(7, types.DoneEvent, &opsTestExpected{
			HasResponse: true,
			Done:        true,
		}),
		newOpsTestInfo(8, types.ErrorEvent, &opsTestExpected{
			Done:       true,
			StatusCode: codepb.Code_UNKNOWN,
		}),
	}
}

func newOpsTestStorageServer(testInfos []*opsTestInfo) (*store.StorageServer, error) {
	var err error
	storageServer := store.NewStorageServer()
	for _, t := range testInfos {
		if t.Session == nil {
			continue
		}
		err = storageServer.StoreSession(t.Session)
		if err != nil {
			return nil, err
		}
		if t.Event == nil {
			continue
		}
		err = storageServer.StoreEvent(t.Session.Name, t.Event)
		if err != nil {
			return nil, err
		}
	}
	return storageServer, nil
}

func TestOperationsServer(t *testing.T) {
	tis := newOpsTestInfos()
	for _, v := range tis {
		ti := v
		t.Run(ti.TestName, func(t *testing.T) {
			t.Parallel()
			var (
				s   *store.StorageServer
				err error
				op  *lrpb.Operation
			)
			s, err = newOpsTestStorageServer(tis)
			if err != nil {
				t.Fatalf("error creating test storage server: %v", err)
			}
			ops := NewOperationsServer(s)
			ctx := context.TODO()
			req := &lrpb.GetOperationRequest{
				Name: ti.OperationName,
			}
			op, err = ops.GetOperation(ctx, req)
			if (err == nil) && ti.Expected.ReturnsError {
				t.Fatalf("expected error, received nil")
			}
			if (err != nil) && !ti.Expected.ReturnsError {
				t.Fatalf("error getting operation: %v", err)
			}
			if err != nil {
				return
			}
			if op == nil {
				t.Fatalf("expected an operation, received nil")
			}
			if op.Done && !ti.Expected.Done {
				t.Fatalf("expected operation done, received not done")
			}
			if !op.Done && ti.Expected.Done {
				t.Fatalf("expected operation not done, received done")
			}
			if op.Metadata == nil {
				t.Fatalf("expected metadata, received nil")
			}
			metadatapb := &svcpb.TestSessionMetadata{}
			err = ptypes.UnmarshalAny(op.Metadata, metadatapb)
			if err != nil {
				t.Fatalf("error unmarshalling metadata: %s", err)
			}
			if metadatapb.LatestEvent != nil {
				eventKind := types.EventKindFromProto(metadatapb.LatestEvent.Kind)
				if ti.Event == nil {
					t.Fatalf(
						"expected no event in metadata, received %s: %q",
						eventKind, metadatapb.LatestEvent.Subject,
					)
				}
				if (metadatapb.LatestEvent.Subject != ti.Event.SubjectName) ||
					(eventKind != ti.Event.Kind) {
					t.Fatalf(
						"metadata event differs, expected %s: %q, received %s: %q",
						ti.Event.Kind, ti.Event.SubjectName,
						eventKind, metadatapb.LatestEvent.Subject,
					)
				}
			}
			if metadatapb.ServiceVersion != Version {
				t.Fatalf("metadata version differs, expected %q, received %q",
					Version, metadatapb.ServiceVersion,
				)
			}
			if !op.Done {
				return
			}
			switch resultType := op.Result.(type) {
			case *lrpb.Operation_Response:
				if !ti.Expected.HasResponse {
					t.Fatalf("expected no response, received response.")
				}
				responsepb := &svcpb.TestSessionResult{}
				err = ptypes.UnmarshalAny(op.GetResponse(), responsepb)
				if err != nil {
					t.Fatalf("error unmarshalling response: %s", err)
				}
				if !bytes.Equal(responsepb.DriverLogs, ti.Event.DriverLogs) {
					t.Fatalf("Driver logs differ, expected %q, received %q",
						string(ti.Event.DriverLogs),
						string(responsepb.DriverLogs),
					)
				}
			case *lrpb.Operation_Error:
				statusCode := codepb.Code(op.GetError().Code)
				if statusCode != ti.Expected.StatusCode {
					t.Fatalf("expected error code %s, received %s",
						ti.Expected.StatusCode, statusCode)
				}
			case nil:
				t.Fatalf("expected result, received nil")
			default:
				t.Fatalf("unexpected result type: %T",
					resultType)
			}
		})
	}
}
