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
	update "github.com/grpc/test-infra/containers/runtime/xds"
	config "github.com/grpc/test-infra/containers/runtime/xds/config"
)

func main() {

	resource := config.TestResource{}

	var nodeID string
	var xdsServerPort uint
	var testListenerPort uint
	var defaultConfigPath string
	var userSuppliedConfigPath string
	var allBackends uint

	// The port that this xDS server listens on
	flag.UintVar(&xdsServerPort, "xdsServerPort", 18000, "xDS management server port, this is where Envoy gets update")

	// Tell Envoy/xDS client to use this Node ID, it is important to match what provided in the bootstrap files
	flag.StringVar(&nodeID, "nodeID", "test_id", "Node ID")

	// Default configuration path
	flag.StringVar(&defaultConfigPath, "DefaultConfigPath", "../config/default_config.json", "The path of default configuration file")

	// User supplied configuration path
	flag.StringVar(&userSuppliedConfigPath, "UserSuppliedConfigPath", "", "The path of user supplied configuration file")

	// Tne cluster name for Envoy obtain configuration from, should match the cluster name in the bootstrap file.
	flag.StringVar(&resource.XDSServerClusterName, "xDSServerClusterName", "xds_cluster", "Tne cluster name for Envoy to obtain configuration, should match the cluster name in the bootstrap file")

	// This sets the gRPC test listener name.
	flag.StringVar(&resource.TestGrpcListenerName, "TestGrpcListenerName", "default_testGrpcListenerName", "This is the gRPC listener's name, should match the server_target_string in xds:///server_target_string")

	// This sets the port that the Envoy listener listens to, this is the port to send traffic if we wish the traffic to go through sidecar
	flag.UintVar(&testListenerPort, "TestListenerPort", 10000, "This sets the port that the test listener listens to")

	// This set the number of the intended backends, the update server will shut down once enough backends are updated
	flag.UintVar(&allBackends, "BackendNumber", 1, "This set the number of the intended backends, the update server will shut down once enough backends are updated")

	flag.Parse()

	resource.TestListenerPort = uint32(testListenerPort)
	// Create a cache
	l := logrus.New()
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Start the endpoint update server
	endpointAddress := make(chan string)
	endpointPort := make(chan uint32)
	backend := config.TestEndpoint{}
	collectedBackends := 0

	go func() {
		update.RunUpdateServer(endpointAddress, endpointPort)
	}()

	for {
		select {
		case backend.TestUpstreamHost = <-endpointAddress:
			backend.TestUpstreamPort = <-endpointPort
			resource.TestEndpoints = append(resource.TestEndpoints, &backend)
			collectedBackends++
			l.Printf("Recieved endpoint address: %v, port: %v", backend.TestUpstreamHost, backend.TestUpstreamPort)
		}
		if collectedBackends == int(allBackends) {
			// Shut down the endpoint update server
			update.StopUpdateServer()
			break
		}
	}

	// Create the snapshot for server
	snapshot, err := resource.GenerateSnapshotFromConfigFiles(defaultConfigPath, userSuppliedConfigPath)
	if err != nil {
		log.Fatalf("fail to create snapshot for xDS server: %v", err)
	}

	// Update endpoint for the snapshot resource
	if err := resource.UpdateEndpoint(snapshot, resource.TestEndpoints); err != nil {
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
