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
	"context"
	"fmt"
	"log"
	"net"

	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
)

// RunxDSServer starts an xDS server at the given port.
func RunxDSServer(ctx context.Context, srv server.Server, port uint, grpcServer *grpc.Server) {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)

	log.Printf("management server listening on %d\n", port)
	if err = grpcServer.Serve(lis); err != nil {
		log.Println(err)
	}
}
