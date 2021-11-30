package main

import (
	"context"
	"flag"
	"os"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"github.com/fsnotify/fsnotify"
	"github.com/grpc/test-infra/tools/xds"
	"github.com/sirupsen/logrus"
)

func main() {

	resource := xds.ServerResource{
		TestServiceClusterName: "test_cluster",
		TestRouteName:          "local_route",
	}

	var nodeID string
	var watchDirectoryFileName string
	var xdsServerPort uint

	// The port that this xDS server listens on
	flag.UintVar(&xdsServerPort, "xdsServerPort", 18000, "xDS management server port, this is where Envoy gets update")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")

	// Tne cluster name for Envoy ontain configuration from, should match the cluster name in the bootstrap file.
	flag.StringVar(&resource.XDSServerClusterName, "XDSServerClusterName", "xds_cluster", "Tne cluster name for Envoy to obtain configuration, should match the cluster name in the bootstrap file")

	// This sets the test listener name
	flag.StringVar(&resource.TestListenerName, "TestListenerName", "test-id", "This is the TestListenerName, should match the server_target_string in xds:///server_target_string")

	// This sets the port that the test listener listens to, this is the port to send traffic if we wish to go through sidecar
	flag.UintVar(&resource.TestListenerPort, "TestListenerPort", 10000, "This sets the port that the test listener listens to. ")

	// Define the directory to watch for Envoy configuration files
	flag.StringVar(&watchDirectoryFileName, "watchDirectoryFileName", "config/", "full path to directory to watch for files")

	flag.Parse()

	// Create a cache
	l := logrus.New()
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Notify channel for file system events
	message := make(chan fsnotify.Event)

	go func() {
		// Watch for file changes
		xds.Watch(watchDirectoryFileName, message)
	}()
	
	select {
	case msg := <-message:
		endpoint, err := xds.ParseYaml(msg.Name)
		if err != nil {
			l.Errorf("yaml parsing error %q", err)
			os.Exit(1)
		}
		resource.TestUpstreamHost = endpoint.Address
		resource.TestListenerPort = endpoint.Port
	}

	// Create the snapshot that we'll serve to Envoy
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
	xds.RunServer(ctx, srv, xdsServerPort)

}
