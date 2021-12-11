package config

import (
	"encoding/json"
	"log"
	"reflect"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	extension "github.com/envoyproxy/go-control-plane/envoy/service/extension/v3"
	runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"


	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/anypb"
)

// CustomSnapshot stores the Resources in similar fashion of cache.Snapshot
type CustomSnapshot struct {
	// only be used for delta xDS
	// https://pkg.go.dev/github.com/envoyproxy/go-control-plane@v0.10.0/pkg/cache/v3#Snapshot
	Version map[string]map[string]string `json:"version,omitempty"`
	// ResourceType: [ResourceName: Resource]
	Resources map[string]map[string]*customResourceWithTTL `json:"resources,omitempty"`
}

// CustomResourceWithTTL
type customResourceWithTTL struct {
	TTL      time.Duration  `json:"ttl,omitempty"`
	Resource customResource `json:"resource,omitempty"`
}
// CustomResource
type customResource struct {
	proto.Message
}

var (
	ResourceTypes = []string{
		resource.ClusterType,
		resource.ListenerType,
		resource.EndpointType,
		resource.ScopedRouteType,
		resource.SecretType,
		resource.ExtensionConfigType,
		resource.RouteType,
		resource.ExtensionConfigType}
)

func (cr *customResource) MarshalJSON() ([]byte, error) {
	anyType, err := anypb.New(cr.Message)
	if err != nil {
		log.Fatalf("fail to marshal proto.any message: %v", err)
	}
	return protojson.MarshalOptions{UseProtoNames: true}.Marshal(anyType)
}

func (cr *customResource) UnmarshalJSON(data []byte) error {
	var a anypb.Any
	if err := protojson.Unmarshal(data, &a); err != nil {
		log.Fatalf("fail to unmarshal proto.any message: %v", err)
	}

	switch a.TypeUrl {
	case resource.EndpointType:
		parsedEndpoint := endpoint.ClusterLoadAssignment{}
		if err := ptypes.UnmarshalAny(&a, &parsedEndpoint); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedEndpoint
	case resource.ClusterType:
		parsedCluster := cluster.Cluster{}
		if err := ptypes.UnmarshalAny(&a, &parsedCluster); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedCluster
	case resource.RouteType:
		parsedRoute := route.RouteConfiguration{}
		if err := ptypes.UnmarshalAny(&a, &parsedRoute); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedRoute
	case resource.ScopedRouteType:
		parsedScopedRoute := route.ScopedRouteConfiguration{}
		if err := ptypes.UnmarshalAny(&a, &parsedScopedRoute); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedScopedRoute
	case resource.ListenerType:
		parsedListener := listener.Listener{}
		if err := ptypes.UnmarshalAny(&a, &parsedListener); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedListener
	case resource.RuntimeType:
		parsedRuntime := runtime.RtdsDummy{}
		if err := ptypes.UnmarshalAny(&a, &parsedRuntime); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedRuntime
	case resource.SecretType:
		parsedSecret := secret.SdsDummy{}
		if err := ptypes.UnmarshalAny(&a, &parsedSecret); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedSecret
	case resource.ExtensionConfigType:
		parsedExtensionConfig := extension.EcdsDummy{}
		if err := ptypes.UnmarshalAny(&a, &parsedExtensionConfig); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Message = &parsedExtensionConfig
	}
	return nil
}

func MarshalSnapshot(snap cache.Snapshot) ([]byte, error) {
	CustomSnapshot := CustomSnapshot{
		Version:   snap.VersionMap,
		Resources: map[string]map[string]*customResourceWithTTL{},
	}

	for _, resourceType := range ResourceTypes {
		resourceWithTTL := map[string]*customResourceWithTTL{}
		for resourceName, resource := range snap.GetResourcesAndTTL(resourceType) {
			resourceWithTTL[resourceName] = &customResourceWithTTL{
				Resource: customResource{
					Message: resource.Resource,
				},
			}
			if !reflect.ValueOf(resource.TTL).IsNil() {
				resourceWithTTL[resourceName].TTL = *resource.TTL
			}

		}
		CustomSnapshot.Resources[resourceType] = resourceWithTTL
	}
	return json.Marshal(CustomSnapshot)
}

func UnMarshalSnapshot(data []byte) (cache.Snapshot, error) {
	customSnapshot := CustomSnapshot{}
	err := json.Unmarshal(data, &customSnapshot)
	if err != nil {
		log.Fatalf("fail to marshal snapshot: %v", err)
	}

	snap := cache.Snapshot{
		VersionMap: customSnapshot.Version,
	}

	for _, resourceType := range ResourceTypes {
		typedResourceItems := map[string]types.ResourceWithTTL{}
		for resourceName, resource := range customSnapshot.Resources[resourceType] {
			typedResourceItems[resourceName] = types.ResourceWithTTL{
				TTL:      &resource.TTL,
				Resource: resource.Resource,
			}
		}
		typ := cache.GetResponseType(resourceType)
		snap.Resources[typ].Items = typedResourceItems
	}

	return snap, nil
}
