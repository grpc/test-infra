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

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

func TestQueueCount(t *testing.T) {
	n := 3 // number of sessions
	q := newQueue(limitlessTracker{})
	sessions := makeSessions(t, n)
	for _, session := range sessions {
		q.Enqueue(session)
	}

	if count := q.Count(); count != n {
		t.Errorf("count did not return accurate number of items, expected %v but got %v", n, count)
	}
}

func TestQueueEnqueue(t *testing.T) {
	n := 3 // number of sessions
	q := newQueue(limitlessTracker{})

	expectedSessions := makeSessions(t, n)
	for _, session := range expectedSessions {
		if err := q.Enqueue(session); err != nil {
			t.Fatalf("encountered unexpected error during session creation: %v", err)
		}
	}

	for i, expected := range expectedSessions {
		actual := q.items[i].session

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("items were missed or added in an incorrect order: %v != %v", actual, expected)
		}
	}
}

func TestQueueDequeue(t *testing.T) {
	n := 3 // number of sessions
	var q *queue
	var sessions []*types.Session

	// test FIFO-order preserved when it can accomodate all sessions
	q = newQueue(limitlessTracker{})

	sessions = makeSessions(t, n)
	for _, session := range sessions {
		q.items = append(q.items, &queueItem{session})
	}

	for i, expected := range sessions {
		got := q.Dequeue()
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("dequeue (iteration %v of %v) was out of order, got %v but expected %v", i+1, n, got, expected)
		}
	}

	// test using ReservationManager
	poolName := "DequeuePool"
	rm := NewReservationManager()
	rm.AddPool(Pool{
		Name:      poolName,
		Available: 3,
		Capacity:  5,
	})

	q = newQueue(rm)

	sessions = makeSessions(t, 3)

	sessions[0].Workers = makeWorkers(t, 5, &poolName)
	q.Enqueue(sessions[0])

	sessions[1].Workers = makeWorkers(t, 3, &poolName)
	q.Enqueue(sessions[1])

	sessions[2].Workers = makeWorkers(t, 1, &poolName)
	q.Enqueue(sessions[2])

	got := q.Dequeue()
	if !reflect.DeepEqual(got, sessions[1]) {
		t.Errorf("dequeue out of order (FIFO then availability), expected %v but got %v", sessions[1], got)
	}

	// test returns nil if queue is empty
	q = newQueue(limitlessTracker{})
	if q.Dequeue() != nil {
		t.Errorf("dequeue returned an object other than <nil> with nothing enqueued")
	}
}

func TestQueueDone(t *testing.T) {
	// check that other workers that need the machines can be dequeued after a done call
	rm := NewReservationManager()
	pool := Pool{
		Name:      "DonePool",
		Available: 7,
		Capacity:  7,
	}
	rm.AddPool(pool)

	q := newQueue(rm)

	fiveWorkers := makeWorkers(t, 5, &pool.Name)
	session1 := types.NewSession(nil, fiveWorkers, nil)
	if err := q.Enqueue(session1); err != nil {
		t.Fatalf("could not enqueue session1 as part of test setup")
	}

	fourWorkers := makeWorkers(t, 4, &pool.Name)
	session2 := types.NewSession(nil, fourWorkers, nil)
	if err := q.Enqueue(session2); err != nil {
		t.Fatalf("could not enqueue session2 as part of test setup")
	}

	twoWorkers := makeWorkers(t, 2, &pool.Name)
	session3 := types.NewSession(nil, twoWorkers, nil)
	if err := q.Enqueue(session3); err != nil {
		t.Fatalf("could not enqueue session3 as part of test setup")
	}

	session := q.Dequeue()
	if session != session1 {
		t.Fatalf("session1 was not dequeued first, test setup failed: %v != %v", session, session1)
	}

	session = q.Dequeue()
	if session != session3 {
		t.Fatalf("session3 was not dequeued second, test setup failed: %v != %v", session, session3)
	}

	q.Done(session1)
	q.Done(session3)

	if session := q.Dequeue(); session != session2 {
		t.Fatalf("session2 not dequeued, indicating Done is likely not increasing available machines")
	}
}
