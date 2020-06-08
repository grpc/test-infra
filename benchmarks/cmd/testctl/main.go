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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"time"

	grpcpb "github.com/grpc/test-infra/proto/grpc/testing"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	svcpb "github.com/grpc/test-infra/benchmarks/proto/scheduling/v1"
	lrpb "google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc"
)

const (
	// Success (exit code 0) shows the command finished without an error.
	Success = 0

	// FlagError (exit code 2) shows the command was unable to run or
	// complete due to the combination or lack of flags.
	FlagError = 2

	// ConnectionError (exit code 3) shows the command could not establish a
	// connection to services over the internet.
	ConnectionError = 3

	// SchedulingError (exit code 4) shows that the test session could not
	// be scheduled to run on the cluster.
	SchedulingError = 4

	// OperationError (exit code 5) shows that the test session was scheduled
	// but there was a problem checking the status of the operation.
	OperationError = 5
)

// ScheduleFlags is the set of flags necessary to schedule test sessions.
type ScheduleFlags struct {
	address    string
	driver     string
	server     string
	driverPool string
	serverPool string
	clientPool string
	clients    clientList
	scenario   scenario
}

// validate ensures that a scenario and driver are provided for the test. If
// they are missing, an error is returned.
func (s *ScheduleFlags) validate() error {
	if s.driver == "" {
		return errors.New("-driver is required to orchestrate the test, but missing")
	}

	if s.scenario.String() == "<nil>" {
		return errors.New("-scenario is required to configure the test, but missing")
	}

	return nil
}

// clientList contains a list of client container images. It implements the
// flag.Value interface, allowing it to be parsed alongside flags with primitive
// types.
type clientList struct {
	clients []string
}

// String returns a string representation of the list of clients.
func (cl *clientList) String() string {
	return fmt.Sprintf("%v", cl.clients)
}

// Set parses a client flag and appends it to the list.
func (cl *clientList) Set(client string) error {
	cl.clients = append(cl.clients, client)
	return nil
}

// scenario wraps the scenario protobuf, implementing the flag.Value interface.
// This allows it to be parsed alongside flags with primitive types.
type scenario struct {
	proto *grpcpb.Scenario
}

// String returns a string representation of the proto.
func (sc *scenario) String() string {
	return fmt.Sprintf("%v", sc.proto)
}

// Set parses the JSON string into a protobuf as the flag is parsed. It returns
// an error is the flag is malformed or cannot be marshaled into a proto.
func (sc *scenario) Set(scenarioJSON string) error {
	if scenarioJSON == "" {
		return errors.New("a valid scenario is required, but missing")
	}

	sc.proto = &grpcpb.Scenario{}
	err := jsonpb.UnmarshalString(scenarioJSON, sc.proto)
	if err != nil {
		return fmt.Errorf("could not parse scenario json: %v", err)
	}

	return nil
}

// connect establishes a connection to a server at a specified address,
// returning a client connection object. If there is a problem connecting or
// the context's deadline is exceeded, an error is returned.
func connect(ctx context.Context, address string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Printf("dialing server at %v\n", address)
	return grpc.DialContext(dialCtx, address, grpc.WithInsecure(),
		grpc.WithBlock(), grpc.WithDisableRetry())
}

// newScheduleRequest uses a ScheduleFlags struct to construct a
// StartTestSessionRequest protobuf.
func newScheduleRequest(flags ScheduleFlags) *svcpb.StartTestSessionRequest {
	var workers []*svcpb.Component
	if flags.server != "" {
		workers = append(workers, &svcpb.Component{
			ContainerImage: flags.server,
			Kind:           svcpb.Component_SERVER,
			Pool:           flags.serverPool,
		})
	}
	for _, client := range flags.clients.clients {
		workers = append(workers, &svcpb.Component{
			ContainerImage: client,
			Kind:           svcpb.Component_CLIENT,
			Pool:           flags.clientPool,
		})
	}

	return &svcpb.StartTestSessionRequest{
		Scenario: flags.scenario.proto,
		Driver: &svcpb.Component{
			ContainerImage: flags.driver,
			Kind:           svcpb.Component_DRIVER,
			Pool:           flags.driverPool,
		},
		Workers: workers,
	}
}

// startSession attempts to create a test session. It returns a longrunning
// operation upon success. If the context's deadline is exceeded or a networking
// problem occurs, an error is returned.
func startSession(ctx context.Context, client svcpb.SchedulingServiceClient, request *svcpb.StartTestSessionRequest) (*lrpb.Operation, error) {
	scheduleCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Printf("scheduling session with test %q\n", request.Scenario.Name)
	return client.StartTestSession(scheduleCtx, request)
}

// awaitSession polls the service for the status of a running operation until it
// is done. If the context's deadline is exceeded or there is a problem getting
// the status of the operation, an error is returned. Otherwise, the result of
// the tests are returned.
func awaitSession(ctx context.Context, operationName string, client lrpb.OperationsClient) (*svcpb.TestSessionResult, error) {
	awaitCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var lastEvent *svcpb.Event

	for {
		operation, err := client.GetOperation(
			awaitCtx,
			&lrpb.GetOperationRequest{Name: operationName},
		)
		if err != nil {
			return nil, fmt.Errorf("could not get operation status: %v", err)
		}

		var metadata svcpb.TestSessionMetadata
		if err := proto.Unmarshal(operation.Metadata.GetValue(), &metadata); err == nil {
			event := metadata.LatestEvent
			timestamp, err := ptypes.Timestamp(event.Time)
			if err != nil {
				return nil, fmt.Errorf("could not marshal timestamp: %v", err)
			}

			if lastEvent == nil || !reflect.DeepEqual(lastEvent, event) {
				fmt.Printf("[%s] [%s] %s\n", timestamp.Format("Jan 2 2006 15:04:05"),
					event.Kind, event.Description)
			}

			lastEvent = event
		}

		if operation.Done {
			var result svcpb.TestSessionResult
			if err := proto.Unmarshal(operation.GetResponse().GetValue(), &result); err != nil {
				return nil, fmt.Errorf("could not marshal test result: %v", err)
			}

			return &result, nil
		}

		time.Sleep(5 * time.Second)
	}
}

// exit logs an error message and terminates the process with the provided
// status code.
func exit(code int, messageFmt string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, messageFmt+"\n", args...)
	os.Exit(code)
}

// Schedule accepts command line arguments and uses them to schedule a test,
// reporting progress as it runs.
func Schedule(args []string) {
	flags := ScheduleFlags{}
	scheduleFlags := flag.NewFlagSet("testctl", flag.ExitOnError)
	scheduleFlags.StringVar(&flags.address, "address", "127.0.0.1:50051", "host and port of the scheduling server")
	scheduleFlags.StringVar(&flags.driver, "driver", "", "container image with a driver for testing")
	scheduleFlags.StringVar(&flags.server, "server", "", "container image with a server for testing")
	scheduleFlags.Var(&flags.clients, "client", "container image with a client for testing")
	scheduleFlags.Var(&flags.scenario, "scenario", "protobuf which configures the test (as a JSON string)")
	scheduleFlags.StringVar(&flags.driverPool, "driverPool", "drivers", "pool of machines where the driver should run")
	scheduleFlags.StringVar(&flags.serverPool, "serverPool", "workers-8core", "pool of machines where the server should run")
	scheduleFlags.StringVar(&flags.clientPool, "clientPool", "workers-8core", "pool of machines where the client should run")
	scheduleFlags.Parse(args)

	if err := flags.validate(); err != nil {
		exit(FlagError, err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := connect(ctx, flags.address)
	if err != nil {
		exit(ConnectionError, "could not connect to server: %v", err)
	}
	defer conn.Close()

	scheduleClient := svcpb.NewSchedulingServiceClient(conn)
	operationsClient := lrpb.NewOperationsClient(conn)

	request := newScheduleRequest(flags)
	operation, err := startSession(ctx, scheduleClient, request)
	if err != nil {
		exit(SchedulingError, "scheduling session failed: %v", err)
	}
	fmt.Printf("%v has been created\n", operation.Name)

	result, err := awaitSession(ctx, operation.Name, operationsClient)
	if err != nil {
		exit(OperationError, "service did not report status of operation: %v", err)
	}
	fmt.Printf("%s\n", result.DriverLogs)
}

func main() {
	Schedule(os.Args[1:])
}
