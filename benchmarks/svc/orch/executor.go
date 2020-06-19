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
	"time"

	"github.com/golang/glog"
	"github.com/grpc/test-infra/benchmarks/svc/store"
	"github.com/grpc/test-infra/benchmarks/svc/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Executors can run a test session by provisioning its components, monitoring
// its health and cleaning up after its termination.
type Executor interface {
	// Execute runs a test session. It accepts a context that can prevent
	// problematic sessions from running indefinitely.
	//
	// An error is returned if there is a problem regarding the test itself.
	// This does not include internal errors that are not specific to the
	// test.
	Execute(context.Context, *types.Session) error
}

type kubeExecutor struct {
	name      string
	watcher   *Watcher
	pcd       podCreateDeleter
	store     store.Store
	session   *types.Session
	eventChan <-chan *PodWatchEvent
}

func (k *kubeExecutor) Execute(ctx context.Context, session *types.Session) error {
	k.setSession(session)
	k.writeEvent(types.AcceptEvent, nil, "kubernetes executor %v assigned session %v", k.name, session.Name)

	k.writeEvent(types.ProvisionEvent, nil, "started provisioning components for session")
	err := k.provision(ctx)
	if err != nil {
		err = fmt.Errorf("failed to provision: %v", err)
		goto endSession
	}

	k.writeEvent(types.RunEvent, nil, "all containers appear healthy, monitoring during tests")
	err = k.monitor(ctx)
	if err != nil {
		err = fmt.Errorf("failed during test: %v", err)
	}

endSession:
	logs, logErr := k.getDriverLogs(ctx)
	if logErr != nil {
		logErr = fmt.Errorf("failed to fetch logs from driver: %v", logErr)
	}

	cleanErr := k.clean(ctx)
	if cleanErr != nil {
		cleanErr = fmt.Errorf("failed to tear-down resources: %v", cleanErr)
	}

	if logErr != nil {
		k.writeEvent(types.InternalErrorEvent, nil, logErr.Error())
	}

	if cleanErr != nil {
		k.writeEvent(types.InternalErrorEvent, logs, cleanErr.Error())
	}

	if err != nil {
		k.writeEvent(types.ErrorEvent, logs, err.Error())
		return err
	}

	k.writeEvent(types.DoneEvent, logs, "session completed, driver container had exit status of 0")
	return nil
}

func (k *kubeExecutor) provision(ctx context.Context) error {
	var components []*types.Component
	var workerIPs []string

	components = append(components, k.session.ServerWorkers()...)
	components = append(components, k.session.ClientWorkers()...)
	components = append(components, k.session.Driver)

	for _, component := range components {
		kind := strings.ToLower(component.Kind.String())

		if component.Kind == types.DriverComponent {
			component.Env["QPS_WORKERS"] = strings.Join(workerIPs, ",")
		}

		glog.Infof("kubeExecutor[%v]: creating %v component %v", k.name, kind, component.Name)

		pod := newSpecBuilder(k.session, component).Pod()
		if _, err := k.pcd.Create(ctx, pod, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("could not create %v component %v: %v", component.Name, kind, err)
		}

		for {
			select {
			case event := <-k.eventChan:
				switch event.Health {
				case Ready:
					ip := event.PodIP
					if len(ip) > 0 {
						host := ip + ":10000"
						workerIPs = append(workerIPs, host)
						glog.V(2).Infof("kubeExecutor[%v]: component %v was assigned IP address %v",
							k.name, event.ComponentName, ip)
						goto componentProvisioned

					}
				case Failed:
					return fmt.Errorf("provision failed due to component %v: %v", event.ComponentName, event.Error)
				default:
					continue
				}
			case <-ctx.Done():
				return fmt.Errorf("provision did not complete before timeout")
			default:
				time.Sleep(1 * time.Second)
			}
		}

	componentProvisioned:
		glog.V(2).Infof("kubeExecutor[%v]: %v component %v is now ready", k.name, kind, component.Name)
	}

	return nil
}

func (k *kubeExecutor) monitor(ctx context.Context) error {
	glog.Infof("kubeExecutor[%v]: monitoring components while session %v runs", k.name, k.session.Name)

	for {
		select {
		case event := <-k.eventChan:
			switch event.Health {
			case Succeeded:
				return nil // no news is good news :)
			case Failed:
				return fmt.Errorf("component %v has failed: %v", event.ComponentName, event.Error)
			}
		case <-ctx.Done():
			return fmt.Errorf("test did not complete before timeout")
		}
	}
}

func (k *kubeExecutor) clean(ctx context.Context) error {
	glog.Infof("kubeExecutor[%v]: deleting components for session %v", k.name, k.session.Name)

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("session-name=%v", k.session.Name),
	}

	err := k.pcd.DeleteCollection(ctx, metav1.DeleteOptions{}, listOpts)
	if err != nil {
		return fmt.Errorf("unable to delete components: %v", err)
	}

	return nil
}

func (k *kubeExecutor) getDriverLogs(ctx context.Context) ([]byte, error) {
	return k.getLogs(ctx, k.session.Driver.Name)
}

func (k *kubeExecutor) getLogs(ctx context.Context, podName string) ([]byte, error) {
	req := k.pcd.GetLogs(podName, &corev1.PodLogOptions{})
	return req.DoRaw(ctx)
}

func (k *kubeExecutor) setSession(session *types.Session) {
	eventChan, _ := k.watcher.Subscribe(session.Name)
	k.eventChan = eventChan
	k.session = session
}

func (k *kubeExecutor) writeEvent(kind types.EventKind, driverLogs []byte, descriptionFmt string, args ...interface{}) {
	description := fmt.Sprintf(descriptionFmt, args...)

	if k.store != nil {
		k.store.StoreEvent(k.session.Name, &types.Event{
			SubjectName: k.session.Name,
			Kind:        kind,
			Time:        time.Now(),
			Description: description,
			DriverLogs:  driverLogs,
		})
	}

	if kind == types.InternalErrorEvent || kind == types.ErrorEvent {
		glog.Errorf("kubeExecutor[%v]: [%v]: %v", k.name, kind, description)
	} else {
		glog.Infof("kubeExecutor[%v]: [%v]: %v", k.name, kind, description)
	}
}
