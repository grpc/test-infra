package xds

import (
	context "context"
	"fmt"
	"log"
	"net"

	config "github.com/grpc/test-infra/containers/runtime/xds/config"
	pb "github.com/grpc/test-infra/proto/testupdater"
	grpc "google.golang.org/grpc"
)

// UpdateServer is used to implement testupdater.TestUpdater.
type UpdateServer struct {
	pb.UnimplementedTestUpdaterServer
	TestInfoChannel chan TestInfo
	srv             *grpc.Server
}

// TestInfo contains the information such as backend's pod address,
// port and the type of the test.
type TestInfo struct {
	Endpoints []*config.TestEndpoint
	TestType  string
}

// UpdateTest implements testupdater.UpdateTest
func (us *UpdateServer) UpdateTest(ctx context.Context, in *pb.TestUpdateRequest) (*pb.TestUpdateReply, error) {
	var testEndpoints []*config.TestEndpoint
	for _, c := range in.GetEndpoints() {
		testEndpoints = append(testEndpoints, &config.TestEndpoint{TestUpstreamHost: c.IpAddress, TestUpstreamPort: c.Port})
		log.Printf("Received endpoint: %v:%v", c.IpAddress, c.Port)
	}
	us.TestInfoChannel <- TestInfo{Endpoints: testEndpoints, TestType: in.TestType}
	return &pb.TestUpdateReply{}, nil
}

// QuitTestUpdateServer stop the UpdateServer.
func (us *UpdateServer) QuitTestUpdateServer(context.Context, *pb.Void) (*pb.Void, error) {
	log.Printf("Shutting down the test update server")
	go us.srv.GracefulStop()

	return &pb.Void{}, nil
}

// RunUpdateServer start a gRPC server listening to test server address and port
func RunUpdateServer(testUpdateChannel chan TestInfo, updatePort uint) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", updatePort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()

	pb.RegisterTestUpdaterServer(srv, &UpdateServer{TestInfoChannel: testUpdateChannel, srv: srv})
	log.Printf("Endpoint update server listening at %v", lis.Addr())
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	log.Print("test listener stopped")
}
