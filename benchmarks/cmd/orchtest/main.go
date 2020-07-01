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
	"flag"
	"log"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"

	"github.com/grpc/test-infra/benchmarks/k8s"
	"github.com/grpc/test-infra/benchmarks/svc/orch"
	"github.com/grpc/test-infra/benchmarks/svc/types"
	grpcpb "github.com/grpc/test-infra/proto/grpc/testing"
)

func main() {
	driver := flag.String("driver", "", "Container image of a driver for testing")
	server := flag.String("server", "", "Container image of a server for testing")
	client := flag.String("client", "", "Container image of a client for testing")
	timeout := flag.Duration("timeout", 5*time.Minute, "Allow the controller to live for this duration")
	scenarioJSON := flag.String("scenarioJSON", "", "Scenario protobuf with test config as a JSON object")
	count := flag.Int("count", 1, "Number of sessions to schedule")

	flag.Parse()
	defer glog.Flush()

	config, set := os.LookupEnv("KUBE_CONFIG_FILE")
	if !set {
		glog.Fatalln("Missing a kube config file, specify its absolute path in the KUBE_CONFIG_FILE env variable.")
	}
	clientset, err := k8s.ConnectWithConfig(config)
	if err != nil {
		glog.Fatalf("Invalid config file specified by the KUBE_CONFIG_FILE env variable, unable to connect: %v", err)
	}

	c, _ := orch.NewController(clientset, nil, nil)
	if err := c.Start(); err != nil {
		panic(err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	defer c.Stop(shutdownCtx)

	go func() {
		for i := 0; i < *count; i++ {
			driver := types.NewComponent(*driver, types.DriverComponent)
			driver.PoolName = "drivers"

			server := types.NewComponent(*server, types.ServerComponent)
			server.PoolName = "workers-8core"

			client := types.NewComponent(*client, types.ClientComponent)
			client.PoolName = "workers-8core"

			c.Schedule(types.NewSession(driver, []*types.Component{server, client}, scenario(*scenarioJSON)))
		}
	}()

	time.Sleep(*timeout)
}

func scenario(scenarioJSON string) *grpcpb.Scenario {
	if len(scenarioJSON) == 0 {
		return nil
	}

	var s grpcpb.Scenario
	if err := jsonpb.UnmarshalString(scenarioJSON, &s); err != nil {
		log.Fatalf("could not parse scenario json: %v", err)
	}
	return &s
}
