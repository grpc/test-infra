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
	"sync"

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

// queue provides a thread-safe FIFO structure that uses a ReservationTracker to only dequeue
// sessions when there are enough available machines.
type queue struct {
	items   []*queueItem
	tracker ReservationTracker
	mux     sync.Mutex
}

// newQueue constructs a queue, given a ReservationTracker.
func newQueue(tracker ReservationTracker) *queue {
	return &queue{
		tracker: tracker,
	}
}

// Enqueue adds a session to the queue.
func (q *queue) Enqueue(session *types.Session) error {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.items = append(q.items, &queueItem{session})
	return nil
}

// Dequeue tries to remove a session from the queue that can run with the current availability. If
// no session can run with the current availability or there are none queued, it returns nil.
//
// If a session is returned, it is the responsibility of the caller to invoke the Done method when
// the session no longer requires its machines.
func (q *queue) Dequeue() *types.Session {
	q.mux.Lock()
	defer q.mux.Unlock()

	for i, item := range q.items {
		session := item.session
		if err := q.tracker.Reserve(session); err == nil {
			// delete item from queue, preserving order of others
			q.items = append(q.items[:i], q.items[i+1:]...)
			return session
		}
	}

	return nil
}

// Done marks the termination of a session, allowing the machines to be used by another session.
func (q *queue) Done(session *types.Session) {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.tracker.Unreserve(session)
}

// Count returns the number of sessions in the queue.
func (q *queue) Count() int {
	q.mux.Lock()
	defer q.mux.Unlock()
	return len(q.items)
}

// queueItem is an internal type which wraps a queued session, designed to allow additional metadata
// and statistics to be coupled with it.
type queueItem struct {
	// session is the waiting session.
	session *types.Session
}
