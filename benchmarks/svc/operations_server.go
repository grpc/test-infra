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
	"strings"

	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	svcpb "github.com/grpc/test-infra/benchmarks/proto/scheduling/v1"
	"github.com/grpc/test-infra/benchmarks/svc/store"
	"github.com/grpc/test-infra/benchmarks/svc/types"
	lrpb "google.golang.org/genproto/googleapis/longrunning"
	codepb "google.golang.org/genproto/googleapis/rpc/code"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
)

// operationNamePrefix is the prefix present in all operation names.
const operationNamePrefix = "operations/"

// OperationsServer implements the google.longrunning.Operations interface
// to provide access to a data Store.
type OperationsServer struct {
	store store.Store
}

// NewOperationsServer constructs a new instance of OperationsServer.
func NewOperationsServer(store store.Store) *OperationsServer {
	return &OperationsServer{
		store: store,
	}
}

// getSessionName converts an operation name to a session name.
// This function returns an error if the operation name is invalid.
func getSessionName(operationName string) (string, error) {
	if !strings.HasPrefix(operationName, operationNamePrefix) {
		return operationName, fmt.Errorf(
			"invalid operation name: %q", operationName)
	}
	return strings.TrimPrefix(operationName, operationNamePrefix), nil
}

// getOperationName converts a session name to an operation name.
func getOperationName(sessionName string) string {
	return fmt.Sprintf("%s%s", operationNamePrefix, sessionName)
}

// isOperationFinished returns true if the operation is finished, and false
// otherwise, based on the latest event received by the operation's session.
func isOperationFinished(e *types.Event) bool {
	// The operation is not finished if there is no event.
	if e == nil {
		return false
	}
	// The operation is finished if the event kind is DoneEvent or an
	// error, otherwise it is not finished.
	switch e.Kind {
	case types.InternalErrorEvent:
		return true
	case types.DoneEvent:
		return true
	case types.ErrorEvent:
		return true
	default:
		return false
	}
}

// isOperationSuccess returns true if the operation is finished and is
// successful, and false otherwise, based on the latest event received by
// the operation's session.
func isOperationSuccess(e *types.Event) bool {
	return (isOperationFinished(e) && e.Kind == types.DoneEvent)
}

// newEventProto constructs an Event proto from an Event object. This
// function returns an error if the event object does not exist or is
// otherwise invalid.
func newEventProto(e *types.Event) (eventpb *svcpb.Event, err error) {
	var time *timestamppb.Timestamp
	if e == nil {
		return
	}
	time, err = ptypes.TimestampProto(e.Time)
	if err != nil {
		err = fmt.Errorf(
			"cannot convert event timestamp to proto: %s",
			err,
		)
		return
	}
	eventpb = &svcpb.Event{
		Subject:     e.SubjectName,
		Kind:        e.Kind.Proto(),
		Description: e.Description,
		Time:        time,
	}
	return
}

// newOperationMetadataProto constructs an operation metadata proto from
// an Event object. The operation metadata proto is of type TestSessionMetadata,
// marshalled to Any. This function returns an error if the metadata proto
// cannot be constructed or cannot be marshalled to Any.
func newOperationMetadataProto(e *types.Event) (metadatapb *anypb.Any, err error) {
	var eventpb *svcpb.Event
	eventpb, err = newEventProto(e)
	if err != nil {
		return
	}
	metadatapb, err = ptypes.MarshalAny(&svcpb.TestSessionMetadata{
		LatestEvent:    eventpb,
		ServiceVersion: Version,
	})
	if err != nil {
		err = fmt.Errorf("could not format test metadata: %s", err)
	}
	return
}

// newOperationResponseProto constructs an operation response proto from an
// Event object and a Session object. The Session object is only needed for the
// creation time. The operation response proto is of type TestSessionResult,
// marshalled to Any. This function returns an error if the response proto
// cannot be constructed or cannot be marshalled to Any.
func newOperationResponseProto(e *types.Event, s *types.Session) (responsepb *anypb.Any, err error) {
	if (e == nil) || (!isOperationFinished(e)) || (s == nil) {
		err = fmt.Errorf("cannot calculate operation response: failed precondition check")
		return
	}
	responsepb, err = ptypes.MarshalAny(&svcpb.TestSessionResult{
		DriverLogs:  e.DriverLogs,
		TimeElapsed: ptypes.DurationProto(s.CreateTime.Sub(e.Time)),
	})
	if err != nil {
		err = fmt.Errorf(
			"could not format test session result into operation response: %v",
			err,
		)
	}
	return
}

// newOperationErrorProto constructs an operation error proto from an Event
// object. The error response proto is of type Status. This function returns
// an error if the error proto cannot be constructed.
func newOperationErrorProto(e *types.Event) (status *statuspb.Status, err error) {
	if (e == nil) || (!isOperationFinished(e)) || isOperationSuccess(e) {
		err = fmt.Errorf("cannot calculate operation error: failed precondition check")
		return
	}
	var statusCode codepb.Code
	switch e.Kind {
	case types.InternalErrorEvent:
		statusCode = codepb.Code_INTERNAL
	default:
		statusCode = codepb.Code_UNKNOWN
	}
	status = &statuspb.Status{
		Code:    int32(statusCode),
		Message: e.Description,
	}
	return
}

// newOperation constructs an operation proto from an Event object and a Session
// object. This function returns en error if the operation proto cannot be
// constructed.
func newOperation(e *types.Event, s *types.Session) (o *lrpb.Operation, err error) {
	if s == nil {
		err = fmt.Errorf("operation must correspond to a session")
		return
	}
	operationName := fmt.Sprintf("%s%s", operationNamePrefix, s.Name)
	var metadatapb *anypb.Any
	metadatapb, err = newOperationMetadataProto(e)
	if err != nil {
		return
	}
	finished := isOperationFinished(e)
	success := isOperationSuccess(e)
	operationpb := &lrpb.Operation{
		Name:     operationName,
		Metadata: metadatapb,
		Done:     finished,
	}
	if success {
		var responsepb *anypb.Any
		responsepb, err = newOperationResponseProto(e, s)
		if err != nil {
			return
		}
		operationpb.Result = &lrpb.Operation_Response{
			Response: responsepb,
		}
	}
	if finished && !success {
		var errorpb *statuspb.Status
		errorpb, err = newOperationErrorProto(e)
		if err != nil {
			return
		}
		operationpb.Result = &lrpb.Operation_Error{
			Error: errorpb,
		}
	}
	o = operationpb
	return
}

// ListOperations returns the list of operations. This function is not
// implemented.
func (o *OperationsServer) ListOperations(ctx context.Context, req *lrpb.ListOperationsRequest) (*lrpb.ListOperationsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetOperation gets an operation. This function returns nil if the operation
// name is invalid or does not correspond to an existing session in the Store.
func (o *OperationsServer) GetOperation(ctx context.Context, req *lrpb.GetOperationRequest) (operation *lrpb.Operation, err error) {
	var (
		sessionName string
		session     *types.Session
		latestEvent *types.Event
	)
	sessionName, err = getSessionName(req.Name)
	// TODO: Should unknown session return an operation?
	if err != nil {
		return
	}
	session = o.store.GetSession(sessionName)
	if session == nil {
		err = fmt.Errorf(
			"operation name does not match an existing session: %q",
			req.Name,
		)
		return
	}
	latestEvent, err = o.store.GetLatestEvent(sessionName)
	if err != nil {
		err = fmt.Errorf("operation not found: %q", req.Name)
		return
	}
	operation, err = newOperation(latestEvent, session)
	return
}

// DeleteOperation deletes an operation. This function is not implemented.
func (o *OperationsServer) DeleteOperation(ctx context.Context, req *lrpb.DeleteOperationRequest) (*emptypb.Empty, error) {
	return nil, fmt.Errorf("not implemented")
}

// CancelOperation cancels an operation. This function is not implemented.
func (o *OperationsServer) CancelOperation(ctx context.Context, req *lrpb.CancelOperationRequest) (*emptypb.Empty, error) {
	return nil, fmt.Errorf("not implemented")
}

// WaitOperation waits for an operation to finish. This function is not
// implemented.
func (o *OperationsServer) WaitOperation(ctx context.Context, req *lrpb.WaitOperationRequest) (*lrpb.Operation, error) {
	return nil, fmt.Errorf("not implemented")
}
