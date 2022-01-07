package xds

import (
	context "context"
	"log"
	"net"

	config "github.com/grpc/test-infra/containers/runtime/xds/config"
	pb "github.com/grpc/test-infra/proto/endpointupdater"
	grpc "google.golang.org/grpc"
)

const (
	updatePort = ":18005"
)

var srv *grpc.Server

// UpdateServer is used to implement endpointupdater.EndpointUpdater.
type UpdateServer struct {
	pb.UnimplementedEndpointUpdaterServer
	EndpointsChannel chan []*config.TestEndpoint
}

// UpdateEndpoint implements endpointupdater.EndpointUpdater
func (us *UpdateServer) UpdateEndpoint(ctx context.Context, in *pb.EndpointUpdaterRequest) (*pb.EndpointUpdaterReply, error) {
	var testEnpoints []*config.TestEndpoint
	for _, c := range in.GetEndpoints() {
		testEnpoints = append(testEnpoints, &config.TestEndpoint{TestUpstreamHost: c.IpAddress, TestUpstreamPort: c.Port})
		log.Printf("Received endpoint: %v:%v", c.IpAddress, c.Port)
	}
	us.EndpointsChannel <- testEnpoints
	return &pb.EndpointUpdaterReply{}, nil
}

// RunUpdateServer start a gRPC server listening to test server address and port
func RunUpdateServer(endpointChannel chan []*config.TestEndpoint) {
	lis, err := net.Listen("tcp", updatePort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv = grpc.NewServer()
	pb.RegisterEndpointUpdaterServer(srv, &UpdateServer{EndpointsChannel: endpointChannel})
	log.Printf("Endpoint update server listening at %v", lis.Addr())
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// StopUpdateServer stops the endpoint update server
func StopUpdateServer() {
	log.Printf("Endpoint updated, shutting down the endpoint update server listening on %v", updatePort)
	srv.Stop()
}
