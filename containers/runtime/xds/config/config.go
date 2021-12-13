package config

import (
	"encoding/json"
	"log"
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

	"google.golang.org/protobuf/types/known/anypb"
)

// CustomSnapshot include a cache.Snapshot for marshal
// and unmarshal purpose
type CustomSnapshot struct {
	cache.Snapshot
}

type customResource struct {
	types.Resource
}

// MarshalJSON is custom MarshalJSON() for CustomSnapshot struct
func (cs CustomSnapshot) MarshalJSON() ([]byte, error) {
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
func (cs *CustomSnapshot) UnmarshalJSON(data []byte) error {
	var values map[string]json.RawMessage
	json.Unmarshal(data, &values)

	// unmarshal VersionMap
	versionMap := make(map[string]map[string]string)
	if err := json.Unmarshal(values["VersionMap"], &versionMap); err != nil {
		log.Fatalf("TODO: error message %v", err)
	}
	cs.VersionMap = versionMap

	// unmarshal data to cache.Resources
	var allResourcesData [types.UnknownType]json.RawMessage
	if err := json.Unmarshal(values["Resources"], &allResourcesData); err != nil {
		log.Fatalf("fail to obtain json.RawMessage of the caches.Resources: %v", err)
	}

	var constructedResources [types.UnknownType]cache.Resources
	for resourceType, typedResourceData := range allResourcesData {
		var typedResources map[string]json.RawMessage
		if err := json.Unmarshal(typedResourceData, &typedResources); err != nil {
			log.Fatalf("fail to obtain json.RawMessage of the caches.Resource: %v", err)
		}

		itemsData := make(map[string]json.RawMessage)
		if err := json.Unmarshal(typedResources["Items"], &itemsData); err != nil {
			log.Fatalf("fail to obtain json.RawMessage of the list of individual types.Resource : %v", err)
		}

		constructedItems := make(map[string]types.ResourceWithTTL)
		for resourceWithTTLName, resourceWithTTLData := range itemsData {
			var resourceWithTTL map[string]json.RawMessage
			if err := json.Unmarshal(resourceWithTTLData, &resourceWithTTL); err != nil {
				log.Fatalf("fail to obtain json.RawMessage of the individual types.ResourceWithTTL : %v", err)
			}

			// get Resource
			customeResource := customResource{}
			if err := json.Unmarshal(resourceWithTTL["Resource"], &customeResource); err != nil {
				log.Fatalf("fail to unmarshal customeResource: %v", err)
			}

			// get TTL
			var ttl time.Duration
			if string(resourceWithTTL["TTL"]) != "null" {
				tempTTL, err := time.ParseDuration(string(resourceWithTTL["TTL"]))
				if err != nil {
					log.Fatalf("fail to parse the TTL duration for individual types.ResourceWithTTL : %v", err)
				}
				ttl = tempTTL
			} else {
				log.Print("No TTL is set")
			}

			// construct the Itemes
			constructedItems[resourceWithTTLName] = types.ResourceWithTTL{
				TTL:      &ttl,
				Resource: customeResource,
			}
		}

		// construct typedResources
		constructedResources[resourceType] = cache.Resources{
			Version: string(typedResources["Version"]),
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
		log.Fatalf("fail to unmarshal proto.any message: %v", err)
	}

	switch a.TypeUrl {
	case resource.EndpointType:
		parsedEndpoint := endpoint.ClusterLoadAssignment{}
		if err := ptypes.UnmarshalAny(&a, &parsedEndpoint); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedEndpoint
	case resource.ClusterType:
		parsedCluster := cluster.Cluster{}
		if err := ptypes.UnmarshalAny(&a, &parsedCluster); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedCluster
	case resource.RouteType:
		parsedRoute := route.RouteConfiguration{}
		if err := ptypes.UnmarshalAny(&a, &parsedRoute); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedRoute
	case resource.ScopedRouteType:
		parsedScopedRoute := route.ScopedRouteConfiguration{}
		if err := ptypes.UnmarshalAny(&a, &parsedScopedRoute); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedScopedRoute
	case resource.ListenerType:
		parsedListener := listener.Listener{}
		if err := ptypes.UnmarshalAny(&a, &parsedListener); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedListener
	case resource.RuntimeType:
		parsedRuntime := runtime.RtdsDummy{}
		if err := ptypes.UnmarshalAny(&a, &parsedRuntime); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedRuntime
	case resource.SecretType:
		parsedSecret := secret.SdsDummy{}
		if err := ptypes.UnmarshalAny(&a, &parsedSecret); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedSecret
	case resource.ExtensionConfigType:
		parsedExtensionConfig := extension.EcdsDummy{}
		if err := ptypes.UnmarshalAny(&a, &parsedExtensionConfig); err != nil {
			log.Fatalf("fail to unmarshal %v resource: %v", resource.EndpointType, err)
		}
		cr.Resource = &parsedExtensionConfig
	}
	return nil
}
