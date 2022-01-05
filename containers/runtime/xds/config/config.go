package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/anypb"
)

// CustomSnapshot include a cache.Snapshot for marshal
// and unmarshal purpose
type customSnapshot struct {
	cache.Snapshot
}

type customResource struct {
	types.Resource
}

// MarshalJSON is custom MarshalJSON() for CustomSnapshot struct
func (cs customSnapshot) MarshalJSON() ([]byte, error) {
	var customResources [types.UnknownType]cache.Resources
	for typeURLNumber, typedResources := range cs.Resources {
		items := make(map[string]types.ResourceWithTTL)
		for resourceName, resourceWithTTL := range typedResources.Items {
			items[resourceName] = types.ResourceWithTTL{
				TTL: resourceWithTTL.TTL,
				Resource: customResource{
					Resource: resourceWithTTL.Resource,
				},
			}
		}
		customResources[typeURLNumber].Items = items
		customResources[typeURLNumber].Version = typedResources.Version
	}
	return json.Marshal(&struct {
		Resources  [types.UnknownType]cache.Resources
		VersionMap map[string]map[string]string
	}{
		Resources:  customResources,
		VersionMap: cs.VersionMap,
	})
}

// MarshalJSON is custom MarshalJSON() for customResource struct
func (cr customResource) MarshalJSON() ([]byte, error) {
	anydata, _ := anypb.New(cr.Resource)
	return protojson.Marshal(anydata)
}

// UnmarshalJSON is custom UnmarshalJSON() for CustomSnapshot struct
func (cs *customSnapshot) UnmarshalJSON(data []byte) error {
	var values map[string]json.RawMessage
	json.Unmarshal(data, &values)

	// unmarshal VersionMap
	versionMap := make(map[string]map[string]string)
	if err := json.Unmarshal(values["VersionMap"], &versionMap); err != nil {
		log.Fatalf("failed to unmarshal VersionMap: %v", err)
	}
	cs.VersionMap = versionMap

	// unmarshal data to cache.Resources
	var allResourcesData [types.UnknownType]json.RawMessage
	if resourcesContent, ok := values["Resources"]; ok {
		if err := json.Unmarshal(resourcesContent, &allResourcesData); err != nil {
			log.Fatalf("failed to obtain json.RawMessage of the caches.Resources: %v", err)
		}
	}

	var constructedResources [types.UnknownType]cache.Resources
	for resourceType, typedResourceData := range allResourcesData {
		var typedResources map[string]json.RawMessage
		if err := json.Unmarshal(typedResourceData, &typedResources); err != nil {
			log.Fatalf("failed to obtain json.RawMessage of the caches.Resource: %v", err)
		}

		itemsData := make(map[string]json.RawMessage)
		if itemsContent, ok := typedResources["Items"]; ok {
			if err := json.Unmarshal(itemsContent, &itemsData); err != nil {
				log.Fatalf("failed to obtain json.RawMessage of the list of individual types.Resource : %v", err)
			}
		}

		constructedItems := make(map[string]types.ResourceWithTTL)
		for resourceWithTTLName, resourceWithTTLData := range itemsData {
			var resourceWithTTL map[string]json.RawMessage
			if err := json.Unmarshal(resourceWithTTLData, &resourceWithTTL); err != nil {
				log.Fatalf("failed to obtain json.RawMessage of the individual types.ResourceWithTTL : %v", err)
			}
			// get Resource
			customeResource := customResource{}
			if resourceContent, ok := resourceWithTTL["Resource"]; ok {
				if err := json.Unmarshal(resourceContent, &customeResource); err != nil {
					log.Fatalf("failed to unmarshal customeResource: %v", err)
				}
			}

			// get TTL
			var ttl *time.Duration
			if ttlContent, ok := resourceWithTTL["TTL"]; ok {
				if string(ttlContent) != "null" {
					var tmpTTL *time.Duration
					err := json.Unmarshal(resourceWithTTL["TTL"], &tmpTTL)
					if err != nil {
						log.Fatalf("failed to unmarshal TTL: %v", err)
					}
					ttl = tmpTTL
				} else {
					log.Printf("No TTL is set for resource: %v", resourceWithTTLName)
				}
			}

			// construct the Items
			constructedItems[resourceWithTTLName] = types.ResourceWithTTL{
				TTL:      ttl,
				Resource: customeResource.Resource,
			}
		}

		// construct typedResources
		var version string
		if versionContent, ok := typedResources["Version"]; ok {
			if err := json.Unmarshal(versionContent, &version); err != nil {
				log.Fatalf("failed to unmarshal version: %v", err)
			}
		}

		constructedResources[resourceType] = cache.Resources{
			Version: version,
			Items:   constructedItems,
		}
	}
	cs.Resources = constructedResources

	return nil
}

// UnmarshalJSON is custom UnmarshalJSON() for customResource struct
func (cr *customResource) UnmarshalJSON(data []byte) error {
	var a anypb.Any
	if err := protojson.Unmarshal(data, &a); err != nil {
		log.Fatalf("failed to unmarshal proto.any message: %v", err)
	}

	switch a.TypeUrl {
	case resource.EndpointType:
		parsedEndpoint := endpoint.ClusterLoadAssignment{}
		if err := anypb.UnmarshalTo(&a, &parsedEndpoint, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		if err := parsedEndpoint.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedEndpoint
	case resource.ClusterType:
		parsedCluster := cluster.Cluster{}
		if err := anypb.UnmarshalTo(&a, &parsedCluster, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.ClusterType, err)
		}
		if err := parsedCluster.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.ClusterType, err)
		}
		cr.Resource = &parsedCluster
	case resource.RouteType:
		parsedRoute := route.RouteConfiguration{}
		if err := anypb.UnmarshalTo(&a, &parsedRoute, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.RouteType, err)
		}
		if err := parsedRoute.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.RouteType, err)
		}
		cr.Resource = &parsedRoute
	case resource.ScopedRouteType:
		parsedScopedRoute := route.ScopedRouteConfiguration{}
		if err := anypb.UnmarshalTo(&a, &parsedScopedRoute, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.ScopedRouteType, err)
		}
		if err := parsedScopedRoute.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.ScopedRouteType, err)
		}
		cr.Resource = &parsedScopedRoute
	case resource.ListenerType:
		parsedListener := listener.Listener{}
		if err := anypb.UnmarshalTo(&a, &parsedListener, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.ListenerType, err)
		}
		// once apiserver is set the socket address can no longer be set, but empty address
		// will fail the validation. TODO: @wanlin31 to figure out a better way
		if err := parsedListener.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.ListenerType, err)
		}
		cr.Resource = &parsedListener
	case resource.RuntimeType:
		parsedRuntime := runtime.Runtime{}
		if err := anypb.UnmarshalTo(&a, &parsedRuntime, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.RuntimeType, err)
		}
		if err := parsedRuntime.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.RuntimeType, err)
		}
		cr.Resource = &parsedRuntime
	case resource.SecretType:
		parsedSecret := secret.Secret{}
		if err := anypb.UnmarshalTo(&a, &parsedSecret, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.SecretType, err)
		}
		if err := parsedSecret.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.SecretType, err)
		}
		cr.Resource = &parsedSecret
	case resource.ExtensionConfigType:
		parsedExtensionConfig := core.TypedExtensionConfig{}
		if err := anypb.UnmarshalTo(&a, &parsedExtensionConfig, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("failed to unmarshal %v resource: %v", resource.ExtensionConfigType, err)
		}
		if err := parsedExtensionConfig.ValidateAll(); err != nil {
			log.Fatalf("failed to validate the parsed %v: %v", resource.ExtensionConfigType, err)
		}
		cr.Resource = &parsedExtensionConfig
	}
	return nil
}

// TestResource provides the names of the resources.
type TestResource struct {
	// xDSServerClusterName is the xDS server's name supplied to Envoy bootstrap file.
	// This field is needed if the RouteConfig or EdsConfig are fetched from the
	// xDS server separately.
	XDSServerClusterName string
	// testServiceClusterName is the name of the upstream clsuter,
	// consisted by the test servers.
	TestServiceClusterName string
	// testRouteName is the route name returned by the test listeners.
	TestRouteName string
	// testGrpcListenerName is only used by gRPC xDS client, must be used as xds:///<listener_name>,
	// API listener is required.
	TestGrpcListenerName string
	// testEnvoyListenerName is only used by Envoy proxy.
	TestEnvoyListenerName string
	// testListenerPort is only used by Envoy, socket listener, traffic will be directly direct to
	// this port.
	TestListenerPort uint32
	// testEndpointName is the name of the cluster. This will be the `service_name
	// in <envoy_v3_api_field_config.cluster.v3.Cluster.EdsClusterConfig.service_name>` value if specified
	// in the cluster.
	TestEndpointName string
	// List of endpoints to load balance to, these will all goes into the same Endpoint resource.
	TestEndpoints []*TestEndpoint
}

// TestEndpoint is the address and the port of the backends.
type TestEndpoint struct {
	// TestUpstreamHost is upstream host address
	TestUpstreamHost string
	// TestUpstreamHost is upstream host port
	TestUpstreamPort uint32
}

func (t *TestResource) validateResource(snap cache.Snapshot) error {
	// gRPC listener name must match match the server_target_string in xds:///server_target_string"
	listenerType := cache.GetResponseType(resource.ListenerType)
	if _, ok := snap.Resources[listenerType].Items[t.TestGrpcListenerName]; !ok {
		log.Fatalf("failed validation of listener resource, please set up gRPC listener with name %v", t.TestGrpcListenerName)
	}

	// Envoy listener's port value is used by routing traffic to Envoy sidecar
	for _, listenerWithTTL := range snap.Resources[listenerType].Items {
		forValidation := listener.Listener{}
		listenerData, err := protojson.Marshal(listenerWithTTL.Resource)
		if err != nil {
			log.Fatalf("failed to validate Envoy listener's port value: %v \n", err)
		}
		if err = protojson.Unmarshal(listenerData, &forValidation); err != nil {
			log.Fatalf("failed to validate Envoy listener's port value: %v \n", err)
		}
		if forValidation.ApiListener != nil && forValidation.Address.GetSocketAddress().GetPortValue() != t.TestListenerPort {
			log.Fatalf("failed to validate Envoy listener's port value: Envoy listener's port value: %v does not match the port that the client target port %v, \n", forValidation.Address.GetSocketAddress().GetPortValue(), t.TestListenerPort)
		}

	}

	// Envoy's dynamic bootstrap file service cluster listed to configure Envoy obtaining configuration from xds server, this cluster name should be listed as the config resource

	// check consistency
	if err := snap.Consistent(); err != nil {
		log.Fatalf("validation failed, snapshpt is inconsistent: %v \n", err)
	}

	return nil
}

// GenerateSnapshotFromConfigFiles takes a default configuration file
// and user supplied configuration to generate a snapshot
func (t *TestResource) GenerateSnapshotFromConfigFiles(defaultConfigPath string, userSuppliedConfigPath string) (cache.Snapshot, error) {
	// read and unmarshal default configuration
	defaultConfigData, err := os.ReadFile(defaultConfigPath)
	if err != nil {
		log.Fatalf("failed to read the default configuration from path (%v): %v", defaultConfigPath, err)
	}

	defaultSnapshot := customSnapshot{}
	if err := json.Unmarshal(defaultConfigData, &defaultSnapshot); err != nil {
		log.Fatalf("failed to unmarshal the default configuration from path (%v): %v \n", defaultConfigPath, err)
	}

	// if not user supplied config, default resource is used
	if _, err := os.Stat(userSuppliedConfigPath); errors.Is(err, os.ErrNotExist) {
		log.Printf("user did not supply configurations for xDS server, use default config at %v", defaultConfigPath)
		if err := t.validateResource(defaultSnapshot.Snapshot); err != nil {
			log.Fatalf("validation failed: %v", err)
		}
		return defaultSnapshot.Snapshot, nil
	}

	// read and unmarshal user supplied configuration
	userSuppliedConfigPathData, err := os.ReadFile(userSuppliedConfigPath)
	if err != nil {
		log.Fatalf("failed to read the user supplied configuration from path (%v): %v \n", userSuppliedConfigPath, err)
	}

	userSuppliedSnapshot := customSnapshot{}
	if err := json.Unmarshal(userSuppliedConfigPathData, &userSuppliedSnapshot); err != nil {
		log.Fatalf("failed to unmarshal the user supplied configuration from path (%v): %v", userSuppliedConfigPath, err)
	}

	// compare default config and user supplied config, if user have supplied
	// the resouce the xDS server will server user supplied config, otherwise
	// the default config will be supplied
	var resources *[types.UnknownType]cache.Resources
	snap := customSnapshot{
		cache.Snapshot{
			Resources: *resources,
		},
	}
	for resourceType := range snap.Resources {
		items := make(map[string]types.ResourceWithTTL)
		// check if user have supplied config for this resource type
		if len(userSuppliedSnapshot.Resources[resourceType].Items) > 0 {
			for resourceName, resourceWithTTL := range userSuppliedSnapshot.Resources[resourceType].Items {
				items[resourceName] = types.ResourceWithTTL{
					Resource: resourceWithTTL.Resource,
					TTL:      resourceWithTTL.TTL,
				}
			}
		} else if len(defaultSnapshot.Resources[resourceType].Items) > 0 {
			// check if default have supplied config for this resource type
			for resourceName, resourceWithTTL := range defaultSnapshot.Resources[resourceType].Items {
				items[resourceName] = types.ResourceWithTTL{
					Resource: resourceWithTTL.Resource,
					TTL:      resourceWithTTL.TTL,
				}
			}
		}
		snap.Resources[resourceType].Items = items

		// provide resource version if supplied by user
		if userSuppliedSnapshot.Resources[resourceType].Version != "" {
			snap.Resources[resourceType].Version = userSuppliedSnapshot.Resources[resourceType].Version
		}
	}

	// validate the snapshot
	if err := t.validateResource(snap.Snapshot); err != nil {
		log.Fatalf("validation failed: %v", err)
	}

	return snap.Snapshot, nil
}

// UpdateEndpoint takes a list of endpoints to updated the Endpoint resources in the snapshot
func (t *TestResource) UpdateEndpoint(snap cache.Snapshot, testEndpoints []*TestEndpoint) error {
	// check endpoint number is correct
	endpointResource := snap.Resources[int(cache.GetResponseType(resource.EndpointType))].Items[t.TestEndpointName].Resource
	data, err := protojson.Marshal(endpointResource)
	if err != nil {
		log.Fatalf("failed to to validate number of the endpoint: %v \n", err)
	}
	endpointService := endpoint.ClusterLoadAssignment{}
	if err := protojson.Unmarshal(data, &endpointService); err != nil {
		log.Fatalf("failed to to validate number of the endpoint: %v \n", err)
	}

	allConfiguredBackends := 0
	for _, localityLbEndpoints := range endpointService.GetEndpoints() {
		allConfiguredBackends += len(localityLbEndpoints.LbEndpoints)
	}

	if len(testEndpoints) != allConfiguredBackends {
		log.Fatalf("number of endpoint supplied from config : %v is different from the actual number of backends: %v \n", allConfiguredBackends, len(testEndpoints))
	}

	// update the endpoints, so far all actual backends are supplied to the same locality group
	for _, eachBackend := range testEndpoints {
		curEndpoint := endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  eachBackend.TestUpstreamHost,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: eachBackend.TestUpstreamPort,
								},
							},
						},
					},
				},
			},
		}
		endpointService.GetEndpoints()[0].LbEndpoints = append(endpointService.GetEndpoints()[0].LbEndpoints, &curEndpoint)
	}
	return nil
}
