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
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"

	"github.com/grpc/test-infra/benchmarks/k8s"
	pb "github.com/grpc/test-infra/benchmarks/proto/scheduling/v1"
	"github.com/grpc/test-infra/benchmarks/svc"
	"github.com/grpc/test-infra/benchmarks/svc/orch"
	"github.com/grpc/test-infra/benchmarks/svc/store"

	lrpb "google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func setupProdEnv() *kubernetes.Clientset {
	clientset, err := k8s.ConnectWithinCluster()
	if err != nil {
		glog.Fatalf("unable to connect to kubernetes API within cluster: %v", err)
	}
	return clientset
}

func setupDevEnv(grpcServer *grpc.Server) *kubernetes.Clientset {
	c, set := os.LookupEnv("KUBE_CONFIG_FILE")
	if !set {
		glog.Fatalf("missing a kube config file, specify its absolute path in the KUBE_CONFIG_FILE env variable")
	}

	clientset, err := k8s.ConnectWithConfig(c)
	if err != nil {
		glog.Fatalf("invalid config file specified by the KUBE_CONFIG_FILE env variable, unable to connect: %v", err)
	}

	glog.Infoln("enabling reflection for grpc_cli; avoid this flag in production")
	reflection.Register(grpcServer)

	return clientset
}

func main() {
	port := flag.Int("port", 50051, "Port to start the service.")
	testTimeout := flag.Duration("testTimeout", 15*time.Minute, "Maximum time tests are allowed to run")
	shutdownTimeout := flag.Duration("shutdownTimeout", 5*time.Minute, "Time alloted to a graceful shutdown.")
	flag.Parse()
	defer glog.Flush()

	grpcServer := grpc.NewServer()
	var clientset *kubernetes.Clientset

	env := os.Getenv("APP_ENV")
	if strings.Compare(env, "production") == 0 {
		glog.Infoln("App environment set to production")
		clientset = setupProdEnv()
	} else {
		glog.Infoln("App environment set to development")
		clientset = setupDevEnv(grpcServer)
	}

	storageServer := store.NewStorageServer()

	controllerOpts := &orch.ControllerOptions{
		TestTimeout: *testTimeout,
	}
	controller, err := orch.NewController(clientset, storageServer, controllerOpts)
	if err != nil {
		glog.Fatalf("could not create a controller: %v", err)
	}

	if err := controller.Start(); err != nil {
		glog.Fatalf("unable to start orchestration controller: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()
	defer controller.Stop(shutdownCtx)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		glog.Fatalf("failed to listen on port %d: %v", *port, err)
	}

	operationsServer := svc.NewOperationsServer(storageServer)

	lrpb.RegisterOperationsServer(grpcServer, operationsServer)

	pb.RegisterSchedulingServiceServer(grpcServer, svc.NewSchedulingServer(
		controller, operationsServer, storageServer,
	))

	glog.Infof("running gRPC server (insecure) on port %d", *port)
	err = grpcServer.Serve(lis)
	if err != nil {
		glog.Fatalf("server unexpectedly crashed: %v", err)
	}
}
