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

package store

import (
	"reflect"
	"testing"

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

type fakeResourceNamer struct {
	resourceName string
}

func (f *fakeResourceNamer) ResourceName() string {
	return f.resourceName
}

func newFakeResourceNamer(resourceName string) types.ResourceNamer {
	return &fakeResourceNamer{
		resourceName: resourceName,
	}
}

func TestStorageServer(t *testing.T) {
	sessions := []*types.Session{
		types.NewSession(nil, make([]*types.Component, 0), nil),
		types.NewSession(nil, make([]*types.Component, 0), nil),
		types.NewSession(nil, make([]*types.Component, 0), nil),
	}
	events := []*types.Event{
		types.NewEvent(newFakeResourceNamer("First Resource"),
			types.AcceptEvent,
			"First Event"),
		types.NewEvent(newFakeResourceNamer("Second Resource"),
			types.RunEvent,
			"Second Event"),
	}
	s := NewStorageServer()
	// Storing a session succeeds.
	err := s.StoreSession(sessions[0])
	if err != nil {
		t.Fatalf("Error storing session: %v", err)
	}
	// Storing a second session succeeds.
	err = s.StoreSession(sessions[1])
	if err != nil {
		t.Fatalf("Error storing session: %v", err)
	}
	// Storing a duplicate session returns an error.
	err = s.StoreSession(sessions[0])
	if err == nil {
		t.Fatalf("Duplicate session should return an error.")
	}
	// Storing an event for en existing session succeeds.
	err = s.StoreEvent(sessions[0].Name, events[0])
	if err != nil {
		t.Fatalf("Error storing event for existing session: %v", err)
	}
	// Storing an event for a non-existing session returns an error.
	err = s.StoreEvent(sessions[2].Name, events[1])
	if err == nil {
		t.Fatalf("Storing event for unknown session should return an error.")
	}
	// Storing a second event for an existing session succeeds.
	err = s.StoreEvent(sessions[0].Name, events[1])
	if err != nil {
		t.Fatalf("Error storing a second event for session: %v", err)
	}
	// Retrieving an existing session succeeds.
	session := s.GetSession(sessions[1].Name)
	if !reflect.DeepEqual(sessions[1], session) {
		t.Fatalf("Error retrieving session: %q", sessions[1].Name)
	}
	// Retrieving a non-existing session returns nil.
	session = s.GetSession(sessions[2].Name)
	if session != nil {
		t.Fatalf(
			"Error retrieving session. Expected nil, got %q",
			session.Name,
		)
	}
	var event *types.Event
	// Getting the latest event from a session that has an event
	// returns an event, and it is the latest.
	event, err = s.GetLatestEvent(sessions[0].Name)
	if err != nil {
		t.Fatalf("Error reading latest event: %v", err)
	}
	if event == nil {
		t.Fatalf("Error reading latest event: event is nil.")
	}
	if !reflect.DeepEqual(event, events[1]) {
		t.Fatalf("Error reading latest event: expected %q, got %q",
			event.SubjectName, events[1].SubjectName)
	}
	// Getting the latest event from a session that has no events
	// returns nil.
	event, err = s.GetLatestEvent(sessions[1].Name)
	if err != nil {
		t.Fatalf("Error reading latest event: %v", err)
	}
	if event != nil {
		t.Fatalf("Error reading latest event: expected nil, got %q",
			event.SubjectName)
	}
	// Getting the latest event from a session that does not exist
	// returns nil and an error.
	event, err = s.GetLatestEvent(sessions[2].Name)
	if err == nil {
		t.Fatalf("Error reading latest event: expected error, got nil.")
	}
	if event != nil {
		t.Fatalf("Error reading latest event: expected nil, got %q",
			event.SubjectName)
	}
	s.DeleteSession(sessions[0].Name)
	// Getting the latest event from a deleted session returns nil
	// and an error.
	event, err = s.GetLatestEvent(sessions[0].Name)
	if err == nil {
		t.Fatalf("Error reading latest event: expected error, got nil.")
	}
	if event != nil {
		t.Fatalf("Error reading latest event: expected nil, got %q",
			event.SubjectName)
	}
	// Storing an event for a deleted session returns an error.
	err = s.StoreEvent(sessions[0].Name, events[0])
	if err == nil {
		t.Fatalf("Storing event for deleted session should return an error.")
	}
}
