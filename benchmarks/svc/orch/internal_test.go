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
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	kubeFake "k8s.io/client-go/kubernetes/fake"
	corev1Fake "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/grpc/test-infra/benchmarks/svc/types"
)

// timeMultiplier allows all test timeouts to be increased or decreased as a whole. This solution
// was recommended by Mitchell Hashimoto's Advanced Testing in Go, see
// <https://about.sourcegraph.com/go/advanced-testing-in-go>.
const timeMultiplier time.Duration = 1

// testContainerImage is a default container image supplied during component construction in tests.
const testContainerImage = "example:latest"

// limitlessTracker allows queue to operate as only a FIFO-queue for easier testing.
type limitlessTracker struct{}

func (lt limitlessTracker) Reserve(session *types.Session) error {
	return nil
}

func (lt limitlessTracker) Unreserve(session *types.Session) error {
	return nil
}

// makeSessions creates the specified number of Session instances. These instances do not have
// components or a scenario.
func makeSessions(t *testing.T, n int) []*types.Session {
	t.Helper()
	var sessions []*types.Session
	for i := 0; i < n; i++ {
		sessions = append(sessions, types.NewSession(nil, nil, nil))
	}
	return sessions
}

// makeWorkers creates a slice of Component instances. The slice will contain exactly 1 server and
// n-1 clients. If a pool is specified, their PoolName field will be assigned to it.
func makeWorkers(t *testing.T, n int, pool *string) []*types.Component {
	t.Helper()
	var components []*types.Component

	if n < 1 {
		return components
	}

	components = append(components, types.NewComponent(testContainerImage, types.ServerComponent))

	for i := n - 1; i > 0; i-- {
		components = append(components, types.NewComponent(testContainerImage, types.ClientComponent))
	}

	if pool != nil {
		for _, c := range components {
			c.PoolName = *pool
		}
	}

	return components
}

// newPodWithSessionName creates an empty kubernetes Pod object. This object is assigned a
// "session-name" label with the specified value.
func newPodWithSessionName(t *testing.T, name string) *corev1.Pod {
	t.Helper()
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"session-name": name,
			},
		},
	}
}

// strUnwrap unwraps a string pointer, returning the string if it is not nil. Otherwise, it returns
// a string with the text "<nil>". This avoids dereference issues in tests.
func strUnwrap(str *string) string {
	val := "<nil>"
	if str != nil {
		return *str
	}

	return val
}

type podWatcherMock struct {
	wi  watch.Interface
	err error
}

func (pwm *podWatcherMock) Watch(listOpts metav1.ListOptions) (watch.Interface, error) {
	if pwm.err != nil {
		return nil, pwm.err
	}

	if pwm.wi == nil {
		return watch.NewRaceFreeFake(), nil
	}

	return pwm.wi, nil
}

type nodeListerMock struct {
	nodes []corev1.Node
	err   error
}

func (nlm *nodeListerMock) List(_ metav1.ListOptions) (*corev1.NodeList, error) {
	if nlm.err != nil {
		return nil, nlm.err
	}

	list := &corev1.NodeList{
		Items: nlm.nodes,
	}

	return list, nil
}

func newKubernetesFake(t *testing.T) *kubeFake.Clientset {
	t.Helper()
	return kubeFake.NewSimpleClientset()
}

func newFakePodInterface(t *testing.T) *corev1Fake.FakePods {
	t.Helper()
	return newKubernetesFake(t).CoreV1().Pods(corev1.NamespaceDefault).(*corev1Fake.FakePods)
}

type executorMock struct {
	err        error
	mux        sync.Mutex
	sideEffect func()
	sessionArg *types.Session
}

func (em *executorMock) Execute(_ context.Context, session *types.Session) error {
	em.mux.Lock()
	defer em.mux.Unlock()

	em.sessionArg = session
	if em.err != nil {
		return em.err
	}

	if em.sideEffect != nil {
		em.sideEffect()
	}

	return nil
}

func (em *executorMock) session() *types.Session {
	em.mux.Lock()
	defer em.mux.Unlock()
	return em.sessionArg
}
