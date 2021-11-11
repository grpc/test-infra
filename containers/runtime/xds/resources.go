package xds

import (
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ServerResource defines the constants to create a snapshot.
type ServerResource struct {
	XDSServerClusterName   string
	TestServiceClusterName string
	TestRouteName          string
	TestListenerName       string
	TestListenerPort       uint
	TestUpstreamHost       string
	TestUpstreamPort       uint
}

func (x *ServerResource) makeCluster() *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 x.TestServiceClusterName,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_LOGICAL_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment:       x.makeEndpoint(),
		Http2ProtocolOptions: &core.Http2ProtocolOptions{},
	}
}

func (x *ServerResource) makeEndpoint() *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: x.TestServiceClusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  x.TestUpstreamHost,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: uint32(x.TestUpstreamPort),
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

func (x *ServerResource) makeRoute() *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: x.TestRouteName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "local_service", //is this name need to match anything?
			Domains: []string{x.TestListenerName},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: x.TestServiceClusterName,
						},
						HostRewriteSpecifier: &route.RouteAction_HostRewriteLiteral{
							HostRewriteLiteral: x.TestUpstreamHost,
						},
					},
				},
			}},
		}},
	}
}

func (x *ServerResource) makeListener() *listener.Listener {
	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    x.makeConfigSource(),
				RouteConfigName: x.TestRouteName,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}
	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: x.TestListenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "localhost", //should be contrained to only listen to localhost?
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(x.TestListenerPort),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

func (x *ServerResource) makeConfigSource() *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       resource.DefaultAPIVersion,
			ApiType:                   core.ApiConfigSource_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: x.XDSServerClusterName},
				},
			}},
		},
	}
	return source
}

// GenerateSnapshot generate a new version of NewSnapshot
// ready to serve.
func (x *ServerResource) GenerateSnapshot() cache.Snapshot {
	snap, _ := cache.NewSnapshot("1",
		map[resource.Type][]types.Resource{
			resource.ClusterType:  {x.makeRoute()},
			resource.RouteType:    {x.makeRoute()},
			resource.ListenerType: {x.makeListener()},
		},
	)
	return snap
}
