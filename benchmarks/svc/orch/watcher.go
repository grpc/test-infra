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
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// Watcher listens for changes to pods in a cluster and forwards these events through channels.
//
// The orch package creates pods for components in a session, labeling each with their session and
// component names. This allows the Watcher to send events related to one session's pods through
// one channel and another session through another channel. The channel that receives events for a
// session is known as the subscriber. It is created and closed through the Subscribe and
// Unsubscribe methods.
//
// Create watcher instances with the NewWatcher constructor, not a literal.
type Watcher struct {
	eventChans      map[string]chan *PodWatchEvent
	eventBufferSize int
	pw              podWatcher
	quit            chan struct{}
	mux             sync.Mutex
	wi              watch.Interface
}

// WatcherOptions overrides the defaults of the watcher, allowing it to be
// configured as needed.
type WatcherOptions struct {
	// EventBufferSize specifies the size of the buffered channel for each
	// session. It allows the watcher to write additional kubernetes events
	// without blocking for reads. It defaults to 32 events.
	EventBufferSize int
}

// NewWatcher creates and prepares a new watcher instance.
func NewWatcher(pw podWatcher, options *WatcherOptions) *Watcher {
	opts := options
	if opts == nil {
		opts = &WatcherOptions{}
	}

	eventBufferSize := opts.EventBufferSize
	if eventBufferSize == 0 {
		eventBufferSize = 32
	}

	return &Watcher{
		eventChans:      make(map[string]chan *PodWatchEvent),
		eventBufferSize: eventBufferSize,
		pw:              pw,
		quit:            make(chan struct{}),
	}
}

// Start creates a new thread that listens for kubernetes events, forwarding them to subscribers.
// It returns an error if there is a problem with kubernetes.
func (w *Watcher) Start() error {
	wi, err := w.pw.Watch(context.Background(), metav1.ListOptions{Watch: true})
	if err != nil {
		return fmt.Errorf("could not start watcher: %v", err)
	}

	w.wi = wi
	go w.watch()
	return nil
}

// Stop prevents additional events from being forwarded to subscribers.
func (w *Watcher) Stop() {
	close(w.quit)

	if w.wi != nil {
		w.wi.Stop()
	}
}

// Subscribe accepts the name of a session and returns a channel or error. The channel will receive
// a list of all events for pods labeled with this session. If there is already a subscriber for the
// session, an error is returned.
func (w *Watcher) Subscribe(sessionName string) (<-chan *PodWatchEvent, error) {
	w.mux.Lock()
	defer w.mux.Unlock()

	_, exists := w.eventChans[sessionName]
	if exists {
		return nil, fmt.Errorf("session %v already has a follower", sessionName)
	}

	eventChan := make(chan *PodWatchEvent, w.eventBufferSize)
	w.eventChans[sessionName] = eventChan
	return eventChan, nil
}

// Unsubscribe accepts the name of a session and prevents the subscriber channel from receiving events
// additional events. If the session has no subscribers, it returns an error.
func (w *Watcher) Unsubscribe(sessionName string) error {
	w.mux.Lock()
	defer w.mux.Unlock()

	eventChan, exists := w.eventChans[sessionName]
	if !exists {
		return fmt.Errorf("cannot unfollow session %v, it does not have a follower", sessionName)
	}

	close(eventChan)
	delete(w.eventChans, sessionName)
	return nil
}

func (w *Watcher) watch() {
	glog.Infoln("watcher: listening for pod events")

	for {
		select {
		case wiEvent := <-w.wi.ResultChan():
			obj := wiEvent.Object
			if obj == nil {
				goto exit
			}

			pod := obj.(*corev1.Pod)
			sessionName, ok := pod.Labels["session-name"]
			if !ok {
				continue // must be a pod that is not for testing
			}

			health, err := w.diagnose(pod)
			event := &PodWatchEvent{
				SessionName:   sessionName,
				ComponentName: pod.Labels["component-name"],
				Health:        health,
				Error:         err,
				Pod:           pod,
				PodIP:         pod.Status.PodIP,
			}
			w.publish(event)
		case <-w.quit:
			goto exit
		}
	}

exit:
	glog.Infof("watcher: terminated gracefully")
}

func (w *Watcher) publish(event *PodWatchEvent) {
	w.mux.Lock()
	defer w.mux.Unlock()

	eventChan := w.eventChans[event.SessionName]
	if eventChan == nil {
		glog.Warningf("watcher: received event for session without subscriber: %v", event)
		return
	}

	if len(eventChan) < cap(eventChan) {
		eventChan <- event
	} else {
		glog.Warningf("watcher: too many events unread in subscriber channel, dropping: %v", event)
	}
}

func (w *Watcher) diagnose(pod *corev1.Pod) (Health, error) {
	status := pod.Status

	if count := len(status.ContainerStatuses); count != 1 {
		return NotReady, fmt.Errorf("pod has %v container statuses, expected 1", count)
	}
	containerStatus := status.ContainerStatuses[0]

	terminationState := containerStatus.LastTerminationState.Terminated
	if terminationState == nil {
		terminationState = containerStatus.State.Terminated
	}

	if terminationState != nil {
		if terminationState.ExitCode == 0 {
			return Succeeded, nil
		}

		return Failed, fmt.Errorf("container terminated unexpectedly: [%v] %v",
			terminationState.Reason, terminationState.Message)
	}

	if waitingState := containerStatus.State.Waiting; waitingState != nil {
		if strings.Compare("CrashLoopBackOff", waitingState.Reason) == 0 {
			return Failed, fmt.Errorf("container crashed: [%v] %v",
				waitingState.Reason, waitingState.Message)
		}
	}

	if containerStatus.State.Running != nil {
		return Ready, nil
	}

	return Unknown, nil
}

// PodWatchEvent is sent to a subscriber on a Watcher whenever there is a change.
type PodWatchEvent struct {
	// SessionName is the name of the session to which the pod belongs.
	SessionName string

	// ComponentName is the name of the component this pod represents.
	ComponentName string

	// Pod is the kubernetes object itself.
	Pod *corev1.Pod

	// PodIP is the pod's IP if available. Otherwise, it is an empty string.
	PodIP string

	// Health is the current health of the pod.
	Health Health

	// Error may provide the error details that led to the failing health.
	Error error
}
