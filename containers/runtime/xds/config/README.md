# Configuration for xDS server

Configuration JSON files are used to generate the xDS configuration for xDS server. The xDS server consumes these configuration and server these configuration to xDS client or Envoy sidecar.

The configuration JSON files are directly unmarshalled to [Snapshot](https://pkg.go.dev/github.com/envoyproxy/go-control-plane@v0.10.0/pkg/cache/v3#Snapshot) struct.

The configuration files should follow the structure in the daufult_config.json file.

* Resources: An array holding the resouces, there are currently 8 types of resources, the index of this array corresponds to the resource type.
* Version: Versions indicate the version of current group of resource. This field is served as key in VersionMap.
* TTL: Optional fields to set TTL for each resource item.
* Items: Items are maps within each resource type, the maps' keys are the individual resource names and the values are the actual resources with an optional TTL
* VersionMap: VersionMap holds the current hash map of all resources in the snapshot. This field should remain nil until it is used. In our use case, we only unmarshal the configuration into Snapshot, the VersionMap remains nil. VersionMap is only to be used with delta xDS.

The source of truth for the fields within each resource in the resource map are their protos.
The 8 types of resources:

* Listener: [config.listener.v3.Listener proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/config/listener/v3/listener.proto#L39)
* Endpoint: [config.endpoint.v3.ClusterLoadAssignment proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/config/endpoint/v3/endpoint.proto#L33)
* Cluster: [config.cluster.v3.Cluster proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/config/cluster/v3/cluster.proto#L47)
* Route: [config.route.v3.RouteConfiguration proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/config/route/v3/route.proto#L26)
* ScopedRoute: [config.route.v3.ScopedRouteConfiguration proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/config/route/v3/scoped_route.proto#L83)
* ExtensionConfig: [config.core.v3.ExtensionConfigSource proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/config/core/v3/extension.proto#L47)
* Secret: [extensions.transport_sockets.tls.v3.Secret proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/extensions/transport_sockets/tls/v3/secret.proto#L40)
* Runtime: [service.runtime.v3.Runtime proto](https://github.com/envoyproxy/envoy/blob/3865cbaec08c5ce44b439eb77e46fe866e37e81a/api/envoy/service/runtime/v3/rtds.proto#L50)

## Default configuration JSON file

The daufult_config.json file contains a piece of configuration that have two listeners with name: `default_testEnvoyListenerName` and `default_testGrpcListenerName`, one Cluster resource, one Route resource and one Endpoint resource. The two listeners are pointing to the same Cluster, eventually the same Endpoint resource.

## User supplied configuration JSON file

If user wish to alter the default configuration, a user defined configuration can be used instead of the default configuration. User can create a configuration json file just like the default_config.json in the same directory with default_config.json, which is `containers/runtime/xds/config` within the test-infra repo. User supplied configurations are updated on top of the default configuration, so user only need to supply the part that they wish to alter, but the user defined the configuration has to follow the same structure with default_config.json.

If the user did not supply any configuration, the default configuration will be used for xDS server.

The user defined configuration can be supplied at the time starting the xDS server, using flag  `-u config/name-of-user-supplied-config.json`.
