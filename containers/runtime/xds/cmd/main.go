package main

import (
	"context"
	"flag"
	"os"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"github.com/grpc/test-infra/containers/runtime/xds"
	update "github.com/grpc/test-infra/containers/runtime/xds"
	"github.com/sirupsen/logrus"
)

func main() {

	resource := xds.ServerResource{
		TestServiceClusterName: "test_cluster",
		TestRouteName:          "test_route",
	}

	var nodeID string
	var xdsServerPort uint

	// The port that this xDS server listens on
	flag.UintVar(&xdsServerPort, "xdsServerPort", 18000, "xDS management server port, this is where Envoy gets update")

	// Tell Envoy/xDS client to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")

	// Tne cluster name for Envoy obtain configuration from, should match the cluster name in the bootstrap file.
	flag.StringVar(&resource.XDSServerClusterName, "XDSServerClusterName", "xds_cluster", "Tne cluster name for Envoy to obtain configuration, should match the cluster name in the bootstrap file")

	// This sets the gRPC test listener name.
	flag.StringVar(&resource.TestGrpcListenerName, "TestGrpcListenerName", "gRPCListener", "This is the gRPC listener's name, should match the server_target_string in xds:///server_target_string")

	// This sets the Envoy test listener name.
	flag.StringVar(&resource.TestEnvoyListenerName, "TestEnvoyListenerName", "envoyListener", "This is the Envoy listener's name")

	// This sets the port that the Envoy listener listens to, this is the port to send traffic if we wish the traffic to go through sidecar
	flag.UintVar(&resource.TestListenerPort, "TestListenerPort", 10000, "This sets the port that the test listener listens to")

	flag.Parse()

	// Create a cache
	l := logrus.New()
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Start the endpoint update server
	endpointAddress := make(chan string)
	endpointPort := make(chan uint32)
	go func() {
		update.RunUpdateServer(endpointAddress, endpointPort)
	}()

	select {
	case resource.TestUpstreamHost = <-endpointAddress:
		resource.TestUpstreamPort = <-endpointPort
		// Shut down the endpoint update server
		update.StopUpdateServer()

		// Create the snapshot to server
		snapshot := resource.GenerateSnapshot()
		if err := snapshot.Consistent(); err != nil {
			l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
			os.Exit(1)
		}
		l.Printf("will serve snapshot %+v", snapshot)

		// Add the snapshot to the cache
		if err := cache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
			l.Errorf("snapshot error %q for %+v", err, snapshot)
			os.Exit(1)
		}
		ctx := context.Background()
		cb := &test.Callbacks{Debug: true}
		srv := server.NewServer(ctx, cache, cb)
		xds.RunxDSServer(ctx, srv, xdsServerPort)
	}
}
