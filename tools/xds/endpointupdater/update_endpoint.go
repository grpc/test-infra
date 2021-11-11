package endpointupdater

import (
	context "context"
	"log"
	"net"

	grpc "google.golang.org/grpc"
)

const (
	port = ":18005"
)

// UpdateServer is used to implement endpointupdater.EndpointUpdater.
type UpdateServer struct {
	UnimplementedEndpointUpdaterServer
	EndpointAddress chan string
}

// UpdateEndpoint implements endpointupdater.EndpointUpdater
func (us *UpdateServer) UpdateEndpoint(ctx context.Context, in *EndpointUpdaterRequest) (*EndpointUpdaterReply, error) {
	log.Printf("Received: %v", in.GetIpAddress())
	us.EndpointAddress <- in.GetIpAddress()
	return &EndpointUpdaterReply{}, nil
}

// RunUpdateServer start a gRPC server listening to test server address 
func RunUpdateServer(targetAddress chan string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RegisterEndpointUpdaterServer(s, &UpdateServer{EndpointAddress: targetAddress})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
