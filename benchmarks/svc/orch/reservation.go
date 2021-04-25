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
	"fmt"

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

// ReservationTracker limits the number of running sessions by considering the number of machines
// that are available.
type ReservationTracker interface {
	// Reserve decreases the number of machines a session requires from the number of available
	// machines. If there are not enough machines available, it returns a PoolAvailabilityError.
	//
	// If the number of machines required exceeds the capacity, Reserve returns a
	// PoolCapacityError. It returns a PoolUnknownError if the session requires an unknown pool.
	Reserve(session *types.Session) error

	// Unreserve increases the number of available machines by the number of machines a session
	// required. Essentially, it reverses the actions of the Reserve function.
	//
	// This method does not ensure that Reserve has been called on the session. If the caller
	// does not invoke Reserve first, the number of available machines may exceed the true
	// capacity.
	//
	// It returns a PoolUnknownError if the session requires an unknown pool.
	Unreserve(session *types.Session) error
}

// ReservationManager contains a set of pools and manages the availability of their machines. It is
// designed to help limit the number of running sessions. It is not thread-safe.
//
// Instances should be created using the NewReservationManager constructor, not a literal.
type ReservationManager struct {
	// pools maps the name of a pool to an instance of the Pool type for quick lookups.
	pools map[string]Pool
}

// NewReservationManager creates a new instance.
func NewReservationManager() *ReservationManager {
	return &ReservationManager{
		pools: make(map[string]Pool),
	}
}

// AddPool adds a pool to the list of pools.
func (rm *ReservationManager) AddPool(pool Pool) {
	rm.pools[pool.Name] = pool
}

// RemovePool removes a pool from the list of pools.
//
// If the pool was never added, it returns a PoolUnknownError.
func (rm *ReservationManager) RemovePool(pool Pool) error {
	name := pool.Name
	if _, ok := rm.pools[name]; !ok {
		return PoolUnknownError{name}
	}

	delete(rm.pools, name)
	return nil
}

// Reserve decreases the number of machines a session requires from the number of available
// machines. If there are not enough machines available, it returns a PoolAvailabilityError.
//
// If the number of machines required exceeds the capacity, Reserve returns a
// PoolCapacityError. It returns a PoolUnknownError if the session requires an unknown pool.
func (rm *ReservationManager) Reserve(session *types.Session) error {
	components := sessionComponents(session)

	machineCounts, err := rm.machineCounts(components)
	if err != nil {
		return err
	}

	fits, err := rm.fits(machineCounts)
	if err != nil {
		return err
	}
	if !fits {
		return PoolAvailabilityError{}
	}

	for name, count := range machineCounts {
		pool := rm.pools[name]
		pool.Available -= count
		rm.pools[name] = pool
	}
	return nil
}

// Unreserve increases the number of available machines by the number of machines a session
// required. Essentially, it reverses the actions of the Reserve function.
//
// This method does not ensure that Reserve has been called on the session. If the caller
// does not invoke Reserve first, the number of available machines may exceed the true
// capacity.
//
// It returns a PoolUnknownError if the session requires an unknown pool.
func (rm *ReservationManager) Unreserve(session *types.Session) error {
	components := sessionComponents(session)

	machineCounts, err := rm.machineCounts(components)
	if err != nil {
		return err
	}

	for name, count := range machineCounts {
		pool := rm.pools[name]
		pool.Available += count
		rm.pools[name] = pool
	}
	return nil
}

// machineCounts returns a map with the name of each pool as the key and the number of machines
// required from the pool as the value. If a component references a pool that was not added to the
// availability instance, it returns a PoolUnknownError.
func (rm *ReservationManager) machineCounts(components []*types.Component) (map[string]int, error) {
	machines := make(map[string]int)

	for poolName := range rm.pools {
		machines[poolName] = 0
	}

	for _, component := range components {
		if c, ok := machines[component.PoolName]; ok {
			machines[component.PoolName] = c + 1
		} else {
			return nil, PoolUnknownError{component.PoolName}
		}
	}

	return machines, nil
}

// fits accepts a map with pool names as keys and number of required machines within the pool as
// values. It returns true if there are enough available resources to schedule. If the number of
// machines required exceeds the capacity of the pools, it returns a PoolCapacityError.
func (rm *ReservationManager) fits(machineCounts map[string]int) (bool, error) {
	for poolName, c := range machineCounts {
		pool := rm.pools[poolName]

		if c > pool.Capacity {
			return false, PoolCapacityError{poolName, c, pool.Capacity}
		}

		if c > pool.Available {
			return false, nil
		}
	}

	return true, nil
}

// PoolAvailabilityError indicates Reserve was called with a session that required a number of
// machines that were not currently available. These machines may become available at another time.
type PoolAvailabilityError struct{}

// Error returns a string representation of the available error message.
func (pae PoolAvailabilityError) Error() string {
	return fmt.Sprintf("not enough machines are available to accommodate the reservation")
}

// PoolCapacityError indicates that a session requires a number of machines which is greater than
// the number of machines in the pool. The session can never be scheduled.
type PoolCapacityError struct {
	// Name is the string identifier for the pool.
	Name string

	// Requested is the number of machines that the session requires.
	Requested int

	// Capacity is the maximum number of machines that the pool can accommodate.
	Capacity int
}

// Error returns a string representation of the capacity error message.
func (pce PoolCapacityError) Error() string {
	return fmt.Sprintf("pool %v has only %v machines, but the session required %v machines",
		pce.Name, pce.Requested, pce.Capacity)
}

// PoolUnknownError indicates that a session component required a pool that was not added on the
// availability instance.
type PoolUnknownError struct {
	// Name is the string identifier for a un-added pool.
	Name string
}

// Error returns a string representation of the unknown error message.
func (pue PoolUnknownError) Error() string {
	return fmt.Sprintf("pool '%v' was referenced, but not added to the availability instance",
		pue.Name)
}

// sessionComponents collects and returns a slice with all of a session's components.
func sessionComponents(session *types.Session) []*types.Component {
	components := []*types.Component{}
	if session.Driver != nil {
		components = append(components, session.Driver)
	}
	for _, w := range session.Workers {
		components = append(components, w)
	}
	return components
}
