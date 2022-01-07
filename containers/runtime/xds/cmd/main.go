package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"

	"github.com/grpc/test-infra/containers/runtime/xds"
	config "github.com/grpc/test-infra/containers/runtime/xds/config"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
)

func main() {

	resource := config.TestResource{}

	var nodeID string
	var xdsServerPort uint
	var testListenerPort uint
	var defaultConfigPath string
	var userSuppliedConfigPath string

	// The port that this xDS server listens on
	flag.UintVar(&xdsServerPort, "xdsServerPort", 18000, "xDS management server port, this is where Envoy gets update")

	// Tell Envoy/xDS client to use this Node ID, it is important to match what provided in the bootstrap files
	flag.StringVar(&nodeID, "nodeID", "test_id", "Node ID")

	// Default configuration path, the path is relative path using ./containers/runtime/xds
	flag.StringVar(&defaultConfigPath, "d", "config/default_config.json", "The path of default configuration file, the path is relative path using ./containers/runtime/xds")

	// User supplied configuration path, the path is relative path using ./containers/runtime/xds
	flag.StringVar(&userSuppliedConfigPath, "u", "", "The path of user supplied configuration file, the path is relative path using ./containers/runtime/xds")

	// Tne cluster name for Envoy obtain configuration from, should match the cluster name in the bootstrap file.
	flag.StringVar(&resource.XDSServerClusterName, "c", "xds_cluster", "Tne cluster name for Envoy to obtain configuration, should match the cluster name in the bootstrap file")

	// This sets the gRPC test listener name.
	flag.StringVar(&resource.TestGrpcListenerName, "g", "default_testGrpcListenerName", "This is the gRPC listener's name, should match the server_target_string in xds:///server_target_string")

	// This sets the port that the Envoy listener listens to, this is the port to send traffic if we wish the traffic to go through sidecar
	flag.UintVar(&testListenerPort, "p", 10000, "This sets the port that the test listener listens to")

	flag.Parse()

	resource.TestListenerPort = uint32(testListenerPort)
	// Create a cache
	l := logrus.New()
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Start the endpoint update server
	endpointChannel := make(chan []*config.TestEndpoint)

	go func() {
		xds.RunUpdateServer(endpointChannel)
	}()

	resource.TestEndpoints = <-endpointChannel
	if resource.TestEndpoints != nil {
		// Create the snapshot for server
		snapshot, err := resource.GenerateSnapshotFromConfigFiles(defaultConfigPath, userSuppliedConfigPath)
		if err != nil {
			log.Fatalf("fail to create snapshot for xDS server: %v", err)
		}

		// Update endpoint for the snapshot resource
		if err := resource.UpdateEndpoint(snapshot); err != nil {
			log.Fatalf("fail to update endpoint for xDS server: %v", err)
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
