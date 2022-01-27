package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"

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
		return errors.Wrapf(err, "failed to unmarshal VersionMap")
	}
	cs.VersionMap = versionMap

	// unmarshal data to cache.Resources
	var allResourcesData [types.UnknownType]json.RawMessage
	if resourcesContent, ok := values["Resources"]; ok {
		if err := json.Unmarshal(resourcesContent, &allResourcesData); err != nil {
			return errors.Wrapf(err, "failed to obtain json.RawMessage of the caches.Resources")
		}
	}

	var constructedResources [types.UnknownType]cache.Resources

	for _, typedResourceData := range allResourcesData {
		var resourceType types.ResponseType
		var typedResources map[string]json.RawMessage

		if typedResourceData == nil {
			continue
		}

		if err := json.Unmarshal(typedResourceData, &typedResources); err != nil {
			return errors.Wrapf(err, "failed to obtain json.RawMessage of the types.Resource")
		}

		itemsData := make(map[string]json.RawMessage)
		if itemsContent, ok := typedResources["Items"]; ok {
			if err := json.Unmarshal(itemsContent, &itemsData); err != nil {
				return errors.Wrapf(err, "failed to obtain json.RawMessage of the list of individual types.Resource")
			}
		}

		constructedItems := make(map[string]types.ResourceWithTTL)
		i := 0
		for resourceWithTTLName, resourceWithTTLData := range itemsData {
			var resourceWithTTL map[string]json.RawMessage
			if err := json.Unmarshal(resourceWithTTLData, &resourceWithTTL); err != nil {
				return errors.Wrapf(err, "failed to obtain json.RawMessage of the individual types.ResourceWithTTL")
			}

			// get Resource
			customeResource := customResource{}
			if resourceContent, ok := resourceWithTTL["Resource"]; ok {
				if i == 0 {
					// check the actual type of the current resource regardles of the order
					var rt anypb.Any
					if err := protojson.Unmarshal(resourceContent, &rt); err != nil {
						return errors.Wrapf(err, "failed to unmarshal proto.any message to determine the resource type")
					}
					resourceType = cache.GetResponseType(rt.TypeUrl)
				}

				if err := json.Unmarshal(resourceContent, &customeResource); err != nil {
					return errors.Wrapf(err, "failed to unmarshal customeResource")
				}
			}

			// get TTL
			var ttl *time.Duration
			if ttlContent, ok := resourceWithTTL["TTL"]; ok {
				if string(ttlContent) != "null" {
					var tmpTTL *time.Duration
					err := json.Unmarshal(resourceWithTTL["TTL"], &tmpTTL)
					if err != nil {
						return errors.Wrapf(err, "failed to unmarshal TTL")
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
			i++
		}

		// skip placeholders
		if len(constructedItems) == 0 {
			continue
		}

		// construct typedResources
		var version string
		if versionContent, ok := typedResources["Version"]; ok {
			if err := json.Unmarshal(versionContent, &version); err != nil {
				return errors.Wrapf(err, "failed to unmarshal version")
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
		return errors.Wrapf(err, "failed to unmarshal proto.any message")
	}

	switch a.TypeUrl {
	case resource.EndpointType:
		parsedEndpoint := endpoint.ClusterLoadAssignment{}
		if err := anypb.UnmarshalTo(&a, &parsedEndpoint, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.EndpointType)
		}
		if err := parsedEndpoint.ValidateAll(); err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.EndpointType)
		}
		cr.Resource = &parsedEndpoint
	case resource.ClusterType:
		parsedCluster := cluster.Cluster{}
		if err := anypb.UnmarshalTo(&a, &parsedCluster, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.ClusterType)
		}
		if err := parsedCluster.ValidateAll(); err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.ClusterType)
		}
		cr.Resource = &parsedCluster
	case resource.RouteType:
		parsedRoute := route.RouteConfiguration{}
		if err := anypb.UnmarshalTo(&a, &parsedRoute, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.RouteType)
		}
		if err := parsedRoute.ValidateAll(); err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.RouteType)
		}
		cr.Resource = &parsedRoute
	case resource.ScopedRouteType:
		parsedScopedRoute := route.ScopedRouteConfiguration{}
		if err := anypb.UnmarshalTo(&a, &parsedScopedRoute, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.ScopedRouteType)
		}
		if err := parsedScopedRoute.ValidateAll(); err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.ScopedRouteType)
		}
		cr.Resource = &parsedScopedRoute
	case resource.ListenerType:
		parsedListener := listener.Listener{}
		if err := anypb.UnmarshalTo(&a, &parsedListener, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.ListenerType)
		}
		// once apiserver is set the socket address can no longer be set, but empty address
		// will fail the validation. TODO: @wanlin31 to figure out a better way
		if err := parsedListener.ValidateAll(); parsedListener.ApiListener == nil && err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.ListenerType)
		}
		cr.Resource = &parsedListener
	case resource.RuntimeType:
		parsedRuntime := runtime.Runtime{}
		if err := anypb.UnmarshalTo(&a, &parsedRuntime, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.RuntimeType)
		}
		if err := parsedRuntime.ValidateAll(); err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.RuntimeType)
		}
		cr.Resource = &parsedRuntime
	case resource.SecretType:
		parsedSecret := secret.Secret{}
		if err := anypb.UnmarshalTo(&a, &parsedSecret, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.SecretType)
		}
		if err := parsedSecret.ValidateAll(); err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.SecretType)
		}
		cr.Resource = &parsedSecret
	case resource.ExtensionConfigType:
		parsedExtensionConfig := core.TypedExtensionConfig{}
		if err := anypb.UnmarshalTo(&a, &parsedExtensionConfig, proto.UnmarshalOptions{}); err != nil {
			return errors.Wrapf(err, "failed to unmarshal resource: %v", resource.ExtensionConfigType)
		}
		if err := parsedExtensionConfig.ValidateAll(); err != nil {
			return errors.Wrapf(err, "failed to validate the parsed resource: %v", resource.ExtensionConfigType)
		}
		cr.Resource = &parsedExtensionConfig
	}
	return nil
}

// TestResource provides the names of the resources.
type TestResource struct {
	// testGrpcListenerName is only used by gRPC xDS client, must be used as xds:///<listener_name>,
	// API listener is required.
	TestGrpcListenerName string
	// testListenerPort is only used by Envoy, socket listener, traffic will be directly direct to
	// this port.
	TestListenerPort uint32
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

// ValidateResource is a method of TestResource, it validates:
// 1. the intended server_target in Non-Proxied test matches at
// lease one listener.
// 2. The port intended to be used in Proxied test is at lease listed
// in one of the socket listeners.
// 3. The configuration generated from the config json file are consistent.
func (t *TestResource) ValidateResource(snap cache.Snapshot) error {
	// gRPC listener name must match match the server_target_string in
	// xds:///server_target_string"
	listenerType := cache.GetResponseType(resource.ListenerType)
	if _, ok := snap.Resources[listenerType].Items[t.TestGrpcListenerName]; !ok {
		return errors.New(fmt.Sprintf("failed validation of listener resource, please set up gRPC listener with name %v", t.TestGrpcListenerName))
	}

	// Envoy listener's port value is used by routing traffic to Envoy sidecar
	for _, listenerWithTTL := range snap.Resources[listenerType].Items {
		forValidation := listener.Listener{}
		listenerData, err := protojson.Marshal(listenerWithTTL.Resource)
		if err != nil {
			return errors.Wrapf(err, "failed to validate Envoy listener's port value")
		}
		if err = protojson.Unmarshal(listenerData, &forValidation); err != nil {
			return errors.Wrapf(err, "failed to validate Envoy listener's port value")
		}
		if forValidation.ApiListener == nil && forValidation.Address.GetSocketAddress().GetPortValue() != t.TestListenerPort {
			return errors.Wrapf(err, "failed to validate Envoy listener's port value: Envoy listener's port value %v does not match the port that the client target port %v, \n", forValidation.Address.GetSocketAddress().GetPortValue(), t.TestListenerPort)
		}
	}
	// check consistency
	if err := snap.Consistent(); err != nil {
		return errors.Wrapf(err, "validation failed, snapshpt is inconsistent")
	}
	return nil
}

// GenerateSnapshotFromConfigFiles takes a default configuration file
// and user supplied configuration to generate a snapshot
func GenerateSnapshotFromConfigFiles(defaultConfigPath string, userSuppliedConfigPath string) (cache.Snapshot, error) {
	// read and unmarshal default configuration
	defaultConfigData, err := os.ReadFile(defaultConfigPath)
	if err != nil {
		return cache.Snapshot{}, errors.Wrapf(err, "failed to read the default configuration from path: %v", defaultConfigPath)
	}

	defaultSnapshot := customSnapshot{}
	if err := json.Unmarshal(defaultConfigData, &defaultSnapshot); err != nil {
		return cache.Snapshot{}, errors.Wrapf(err, "failed to unmarshal the default configuration from path %v", defaultConfigPath)
	}

	// if not user supplied config, default resource is used
	if _, err := os.Stat(userSuppliedConfigPath); errors.Is(err, os.ErrNotExist) {
		log.Printf("user did not supply configurations for xDS server, use default config at %v", defaultConfigPath)
		return defaultSnapshot.Snapshot, nil
	}

	// read and unmarshal user supplied configuration
	userSuppliedConfigPathData, err := os.ReadFile(userSuppliedConfigPath)
	if err != nil {
		return cache.Snapshot{}, errors.Wrapf(err, "failed to read the user supplied configuration from path: %v", userSuppliedConfigPath)
	}

	userSuppliedSnapshot := customSnapshot{}
	if err := json.Unmarshal(userSuppliedConfigPathData, &userSuppliedSnapshot); err != nil {
		return cache.Snapshot{}, errors.Wrapf(err, "failed to unmarshal the user supplied configuration from path: %v", userSuppliedConfigPath)
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

	return snap.Snapshot, nil
}

// UpdateEndpoint takes a list of endpoints to updated the Endpoint resources in the snapshot
func (t *TestResource) UpdateEndpoint(snap *cache.Snapshot) error {
	// currently we only support one cluster, get the endpointName from the cluster resource
	// break after the d first cluster
	for _, clusterResource := range snap.Resources[int(cache.GetResponseType(resource.ClusterType))].Items {
		// get the cluster resource to obtain the endpoint name associated with the cluster
		clusterData, err := protojson.Marshal(clusterResource.Resource)
		if err != nil {
			return err
		}
		curCluster := cluster.Cluster{}
		if err := protojson.Unmarshal(clusterData, &curCluster); err != nil {
			return err
		}

		// check if endpoint number is correct
		endpointResource := snap.Resources[int(cache.GetResponseType(resource.EndpointType))].Items[curCluster.GetEdsClusterConfig().ServiceName].Resource
		endpointData, err := protojson.Marshal(endpointResource)
		if err != nil {
			return err
		}
		endpointService := endpoint.ClusterLoadAssignment{}
		if err := protojson.Unmarshal(endpointData, &endpointService); err != nil {
			return err
		}

		allConfiguredBackends := 0
		for _, localityLbEndpoints := range endpointService.GetEndpoints() {
			allConfiguredBackends += len(localityLbEndpoints.LbEndpoints)
		}

		if len(t.TestEndpoints) != allConfiguredBackends {
			return errors.New(fmt.Sprintf("number of endpoint supplied from config : %v is different from the actual number of backends: %v \n", allConfiguredBackends, len(t.TestEndpoints)))
		}

		// update the endpoints, so far all actual backends are supplied to the same locality group
		updatedEndpoints := []*endpoint.LbEndpoint{}
		for _, eachBackend := range t.TestEndpoints {
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
			updatedEndpoints = append(updatedEndpoints, &curEndpoint)
		}
		endpointService.GetEndpoints()[0].LbEndpoints = updatedEndpoints
		snap.Resources[int(cache.GetResponseType(resource.EndpointType))].Items[curCluster.GetEdsClusterConfig().ServiceName] = types.ResourceWithTTL{Resource: &endpointService}
		break
	}
	return nil
}

// SocketListenerOnly takes a pointer of a snapshot, and returns only the socket listeners.
// This function is used for Proxied test since api_listneners which are used for the
// non-proxed test can not be validated by Envoy causing the entire resources slices not to be
// registed.
func SocketListenerOnly(snap *cache.Snapshot) error {
	listenerResponseType := cache.GetResponseType(resource.ListenerType)
	listeners := snap.Resources[int(listenerResponseType)]
	socketListenerOnly := make(map[string]types.ResourceWithTTL)
	for listenerName, listenerResource := range listeners.Items {
		listenerData, err := protojson.Marshal(listenerResource.Resource)
		if err != nil {
			return err
		}
		curlistener := listener.Listener{}
		if err := protojson.Unmarshal(listenerData, &curlistener); err != nil {
			return err
		}
		if curlistener.GetApiListener() == nil && curlistener.GetAddress().Address != nil {
			socketListenerOnly[listenerName] = types.ResourceWithTTL{
				Resource: &curlistener,
				TTL:      listenerResource.TTL,
			}
		}
	}
	snap.Resources[int(listenerResponseType)] = cache.Resources{
		Version: listeners.Version,
		Items:   socketListenerOnly,
	}

	return nil
}
