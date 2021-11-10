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
	log.Printf("Received: %v:%v", in.GetIpAddress(), in.GetPort())
	us.EndpointAddress <- in.GetIpAddress()
	return &EndpointUpdaterReply{}, nil
}

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
