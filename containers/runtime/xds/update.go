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

// UpdateServer is used to implement endpointupdater.EndpointUpdater.
type UpdateServer struct {
	pb.UnimplementedEndpointUpdaterServer
	EndpointsChannel chan []*config.TestEndpoint
	srv              *grpc.Server
}

// UpdateEndpoint implements endpointupdater.EndpointUpdater
func (us *UpdateServer) UpdateEndpoint(ctx context.Context, in *pb.EndpointUpdaterRequest) (*pb.EndpointUpdaterReply, error) {
	var testEndpoints []*config.TestEndpoint
	for _, c := range in.GetEndpoints() {
		testEndpoints = append(testEndpoints, &config.TestEndpoint{TestUpstreamHost: c.IpAddress, TestUpstreamPort: c.Port})
		log.Printf("Received endpoint: %v:%v", c.IpAddress, c.Port)
	}
	us.EndpointsChannel <- testEndpoints
	return &pb.EndpointUpdaterReply{}, nil
}

// QuitEndpointUpdateServer stop the EndpointUpdateServer.
func (us *UpdateServer) QuitEndpointUpdateServer(context.Context, *pb.Void) (*pb.Void, error) {
	log.Printf("Shutting down the endpoint update server listening on %v", updatePort)
	go func() {
		us.srv.GracefulStop()
	}()

	return &pb.Void{}, nil
}

// RunUpdateServer start a gRPC server listening to test server address and port
func RunUpdateServer(endpointChannel chan []*config.TestEndpoint) {
	lis, err := net.Listen("tcp", updatePort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()

	pb.RegisterEndpointUpdaterServer(srv, &UpdateServer{EndpointsChannel: endpointChannel, srv: srv})
	log.Printf("Endpoint update server listening at %v", lis.Addr())
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	log.Print("listener aborted")
}
