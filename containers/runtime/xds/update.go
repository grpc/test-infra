package xds

import (
	context "context"
	"log"
	"net"

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
	EndpointAddress chan string
	EndpointPort    chan uint32
}

// UpdateEndpoint implements endpointupdater.EndpointUpdater
func (us *UpdateServer) UpdateEndpoint(ctx context.Context, in *pb.EndpointUpdaterRequest) (*pb.EndpointUpdaterReply, error) {
	us.EndpointAddress <- in.GetIpAddress()
	us.EndpointPort <- in.GetPort()
	log.Printf("Received endpoint: %v:%v", us.EndpointAddress, us.EndpointPort)
	return &pb.EndpointUpdaterReply{}, nil
}

// RunUpdateServer start a gRPC server listening to test server address and port
func RunUpdateServer(testHostAddress chan string, testPort chan uint32) {
	lis, err := net.Listen("tcp", updatePort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv = grpc.NewServer()
	pb.RegisterEndpointUpdaterServer(srv, &UpdateServer{EndpointAddress: testHostAddress, EndpointPort: testPort})
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
