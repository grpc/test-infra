package main

import (
	"context"
	"flag"

	"os"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
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
	var sidecarListenerPort uint
	var defaultConfigPath string
	var customConfigPath string
	var endpointUpdaterPort uint
	var validationOnly bool

	// The port that this xDS server listens on
	flag.UintVar(&xdsServerPort, "xds-server-port", 18000, "xDS management server port, this is where Envoy/gRPC client gets update")

	// The port that endpoint updater server listens on
	flag.UintVar(&endpointUpdaterPort, "endpoint-update-port", 18005, "endpointUpdater server port, this is where endpoint updater gets the IP and port for test servers")

	// Tell Envoy/xDS client to use this Node ID, it is important to match what provided in the bootstrap files
	flag.StringVar(&nodeID, "node-ID", "test_id", "Node ID")

	// Default configuration path, the path is relative path using ./containers/runtime/xds
	flag.StringVar(&defaultConfigPath, "default-config-path", "containers/runtime/xds/config/default_config.json", "The path of default configuration file, the path is relative path the root of test-infra repo")

	// User supplied configuration path, the path is relative path using ./containers/runtime/xds
	flag.StringVar(&customConfigPath, "custom-config-path", "custom-config-path", "The path of user supplied configuration file, the path is relative path the root of test-infra repo")

	// This sets the gRPC test listener name.
	flag.StringVar(&resource.TestGrpcListenerName, "psm-target-string", "default_testGrpcListenerName", "This field is for validation only, the gRPC listener's name, should match the server_target_string in xds:///server_target_string")

	// This sets the port that the Envoy listener listens to, this is the port to send traffic if we wish the traffic to go through sidecar
	flag.UintVar(&sidecarListenerPort, "sidecar-listener-port", 10000, "This field is for validation only, this is port that the sidecar test listener listens to")

	// This sets if running validation only
	flag.BoolVar(&validationOnly, "validation-only", false, "This sets if we are running for the validation only")

	flag.Parse()

	resource.TestListenerPort = uint32(sidecarListenerPort)

	l := log.NewDefaultLogger()

	// Create and validate the configuration of the xDS server first
	snapshot, err := resource.GenerateSnapshotFromConfigFiles(defaultConfigPath, customConfigPath)
	if err != nil {
		l.Errorf("fail to create snapshot for xDS server: %v", err)
	}
	l.Infof("xDS server resource snapshot is generated successfully")

	if validationOnly {
		return
	}

	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Start the endpoint update server
	endpointChannel := make(chan []*config.TestEndpoint)

	go xds.RunUpdateServer(endpointChannel, endpointUpdaterPort)

	resource.TestEndpoints = <-endpointChannel
	if resource.TestEndpoints != nil {
		// Update endpoint for the snapshot resource
		if err := resource.UpdateEndpoint(snapshot); err != nil {
			l.Errorf("fail to update endpoint for xDS server: %v", err)
		}

		l.Infof("will serve snapshot %+v", snapshot)

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
