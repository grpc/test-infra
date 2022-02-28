/*
Copyright 2021 gRPC authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"google.golang.org/grpc"

	grpcv1config "github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/containers/runtime/xds-server"
	config "github.com/grpc/test-infra/containers/runtime/xds-server/config"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
)

func main() {

	var nodeID string
	var xdsServerPort uint
	var defaultConfigPath string
	var customConfigPath string
	var testUpdatePort uint
	var validationOnly bool
	var pathToBootstrap string

	// The port that this xDS server listens on
	flag.UintVar(&xdsServerPort, "xds-server-port", 18000, "xDS management server port, this is where Envoy/gRPC client gets update")

	// The port that endpoint updater server listens on
	flag.UintVar(&testUpdatePort, "test-update-port", grpcv1config.ServerUpdatePort, "test update server port, this is where test updater pass the endpoints and test type to xds server")

	// Tell Envoy/xDS client to use this Node ID, it is important to match what provided in the bootstrap files
	flag.StringVar(&nodeID, "node-ID", "test_id", "Node ID")

	// Default configuration path, the path is relative path using ./containers/runtime/xds
	flag.StringVar(&defaultConfigPath, "default-config-path", "containers/runtime/xds/config/default_config.json", "The path of default configuration file, the path is relative path the root of test-infra repo")

	// User supplied configuration path, the path is relative path using ./containers/runtime/xds
	flag.StringVar(&customConfigPath, "custom-config-path", "custom-config-path", "The path of user supplied configuration file, the path is relative path the root of test-infra repo")

	// This sets if running validation only
	flag.BoolVar(&validationOnly, "validate-only", false, "This sets if we are running for the validation only")

	// This set the path to the original bootstrap file in xds container image, if not set the bootstrap will not be moved
	flag.StringVar(&pathToBootstrap, "path-to-bootstrap", "", "This sets the original path to bootstrap")

	flag.Parse()

	l := xds.Logger{}

	// Create and validate the configuration of the xDS server first
	snapshot, err := config.GenerateSnapshotFromConfigFiles(defaultConfigPath, customConfigPath)
	if err != nil {
		l.Errorf("fail to generate resource snapshot from configuration json file for xDS server: %v", err)
	}

	// validate the snapshot
	if snapshot.Consistent(); err != nil {
		l.Errorf("fail to validate the generated snapshot for xDS server: %v", err)
	}

	l.Infof("xDS server resource snapshot is generated successfully")

	if validationOnly {
		return
	}
	// Move the bootstrap file for proxyless client. The bootstrap files need to be
	// accessible to the proxyless client, this logic helps move the bootstrap.json stored
	// in xds server image to a shared volume between xds server and proxyless client at
	// /bootstrap/bootstrap.json. From the root of the test-infra
	// repo, the bootstrap file was originally stored in the path of
	// contaners/runtime/xds/bootstrap.json, user can provide other path within
	// test-infra, any changes related to this requires building a new xds server image.
	if pathToBootstrap != "" {
		bootstrapBytes, err := ioutil.ReadFile(pathToBootstrap)
		if err != nil {
			l.Errorf("fail to read bootstrap: %v", err)
		}
		//Copy all the contents to the desitination file
		err = ioutil.WriteFile(fmt.Sprintf("%v/bootstrap.json", "/bootstrap"), bootstrapBytes, 0755)
		if err != nil {
			l.Errorf("fail to output bootstrap.json to /bootstrap: %v", err)
		}
		l.Infof("bootstrap file for non-proxied clients are moved from %v to %v/bootstrap.json successfully", pathToBootstrap, "/bootstrap")
	}

	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, l)

	// Start the endpoint update server
	testChannel := make(chan xds.TestInfo)

	// Don't need to handle this server since if the test was terminated
	// at this stage there must be something wrong with the test, no need
	// for grace termination.
	go xds.RunUpdateServer(testChannel, testUpdatePort, &snapshot)

	var testInfo xds.TestInfo
	testInfo, ok := <-testChannel
	if ok {
		// Update test endpoint and type for the snapshot resource
		endpoints := testInfo.Endpoints
		if err := config.UpdateEndpoint(&snapshot, endpoints); err != nil {
			l.Errorf("fail to update endpoint for xDS server: %v", err)
		}

		// Check the type of the test
		if testInfo.IsProxied {
			l.Infof("running a proxied test, only leave socket listeners for validation reason, api_listeners are not presented to proxies")
			if err := config.IncludeSocketListenerOnly(&snapshot); err != nil {
				l.Errorf("fail to filter listener based on test type: %v", err)
			}
			if err := snapshot.Consistent(); err != nil {
				l.Errorf("fail to validate snapshot after leave only socket listeners: %v", err)
			}
		}

		l.Infof("will serve snapshot %+v", snapshot)

		// Add the snapshot to the cache
		if err := cache.SetSnapshot(context.Background(), nodeID, snapshot); err != nil {
			l.Errorf("snapshot error %q for %+v", err, snapshot)
		}
		ctx := context.Background()
		cb := &test.Callbacks{Debug: true}
		srv := server.NewServer(ctx, cache, cb)

		grpcServer := grpc.NewServer()

		// This is to gracefully shutdown the xds server
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM)
		go func() {
			sig, ok := <-sigs
			l.Infof("test complete, gracefully shutting down xds server, shutting down on %v", sig)
			if ok {
				grpcServer.GracefulStop()
			}
		}()

		xds.RunxDSServer(ctx, srv, xdsServerPort, grpcServer)
	}
}
