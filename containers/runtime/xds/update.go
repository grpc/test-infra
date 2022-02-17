/*
Copyright 2021 gRPC authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package xds

import (
	context "context"
	"fmt"
	"log"
	"net"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	config "github.com/grpc/test-infra/containers/runtime/xds/config"
	pb "github.com/grpc/test-infra/proto/endpointupdater"
	grpc "google.golang.org/grpc"
)

// UpdateServer is used to implement testupdater.TestUpdater.
type UpdateServer struct {
	pb.UnimplementedTestUpdaterServer
	TestInfoChannel chan TestInfo
	Srv             *grpc.Server
	Snapshot        *cache.Snapshot
}

// TestInfo contains the information such as backend's pod address,
// port and the type of the test.
type TestInfo struct {
	Endpoints []config.TestEndpoint
	IsProxied bool
}

// UpdateTest implements testupdater.UpdateTest
func (us *UpdateServer) UpdateTest(ctx context.Context, in *pb.TestUpdateRequest) (*pb.TestUpdateReply, error) {
	var testEndpoints []config.TestEndpoint

	log.Printf("Running proxied test: %v", in.IsProxied)

	for _, c := range in.GetEndpoints() {
		testEndpoints = append(testEndpoints, config.TestEndpoint{TestUpstreamHost: c.IpAddress, TestUpstreamPort: c.Port})
		log.Printf("Received endpoint: %v:%v", c.IpAddress, c.Port)
	}
	us.TestInfoChannel <- TestInfo{Endpoints: testEndpoints, IsProxied: in.IsProxied}

	response := &pb.TestUpdateReply{}
	if in.IsProxied {
		target, err := config.ConstructProxiedTestTarget(us.Snapshot)
		if err != nil {
			return nil, err
		}
		response.PsmServerTargetOverride = target
	} else {
		target, err := config.ConstructProxylessTestTarget(us.Snapshot)
		if err != nil {
			return nil, err
		}
		response.PsmServerTargetOverride = target
	}

	return response, nil
}

// QuitTestUpdateServer stop the UpdateServer.
func (us *UpdateServer) QuitTestUpdateServer(context.Context, *pb.Void) (*pb.Void, error) {
	log.Printf("Shutting down the test update server")
	go us.Srv.GracefulStop()

	return &pb.Void{}, nil
}

// RunUpdateServer start a gRPC server listening to test server address and port
func RunUpdateServer(testUpdateChannel chan TestInfo, updatePort uint, snapshot *cache.Snapshot) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", updatePort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()

	pb.RegisterTestUpdaterServer(srv, &UpdateServer{TestInfoChannel: testUpdateChannel, Srv: srv, Snapshot: snapshot})
	log.Printf("Endpoint update server listening at %v", lis.Addr())
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	log.Print("test update server stopped")
}
