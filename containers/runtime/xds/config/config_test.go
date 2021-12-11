package config

import (
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	extension "github.com/envoyproxy/go-control-plane/envoy/service/extension/v3"
	runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	v3httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func makeGrpcHTTPListener() *listener.Listener {
	a, _ := anypb.New(&v3routerpb.Router{})

	hcm, _ := anypb.New(&v3httppb.HttpConnectionManager{
		RouteSpecifier: &v3httppb.HttpConnectionManager_Rds{Rds: &v3httppb.Rds{
			ConfigSource: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{Ads: &core.AggregatedConfigSource{}},
			},
			RouteConfigName: "s.TestRouteName",
		}},
		// router fields are unused by grpc
		HttpFilters: []*v3httppb.HttpFilter{{
			Name: "router",
			ConfigType: &v3httppb.HttpFilter_TypedConfig{
				TypedConfig: a,
			},
		}},
	},
	)
	return &listener.Listener{
		Name:        "s.TestGrpcListenerName",
		ApiListener: &listener.ApiListener{ApiListener: hcm},
		FilterChains: []*listener.FilterChain{{
			Name: "filter-chain-name",
			Filters: []*listener.Filter{{
				Name:       wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{TypedConfig: hcm},
			}},
		}},
	}
}

func makeEnvoyHTTPListener() *listener.Listener {
	// HTTP filter configuration
	manager := &v3httppb.HttpConnectionManager{
		CodecType:  v3httppb.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &v3httppb.HttpConnectionManager_Rds{
			Rds: &v3httppb.Rds{
				ConfigSource: &core.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
						ApiConfigSource: &core.ApiConfigSource{
							TransportApiVersion:       resource.DefaultAPIVersion,
							ApiType:                   core.ApiConfigSource_GRPC,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*core.GrpcService{{
								TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
								},
							}},
						},
					}},
				RouteConfigName: "s.TestRouteName",
			},
		},
		HttpFilters: []*v3httppb.HttpFilter{{
			Name: wellknown.Router,
		}},
	}
	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: "s.TestEnvoyListenerName",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(1234),
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

func prepareTestResource() cache.Snapshot {
	endpointTest := &endpoint.ClusterLoadAssignment{
		ClusterName: "test",
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			Locality: &core.Locality{SubZone: "subzone"},
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  "100:200:300:400",
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: uint32(1234),
									},
								},
							},
						},
					},
				},
			}},
			LoadBalancingWeight: &wrapperspb.UInt32Value{Value: 1},
			Priority:            0,
		}},
	}

	clusterTest := &cluster.Cluster{
		Name:                 "wanlin",
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
			ServiceName: "wanlin",
		},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		Http2ProtocolOptions: &core.Http2ProtocolOptions{},
	}
	routeTest := &route.RouteConfiguration{
		Name: "wanlin",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "example_virtual_host",
			Domains: []string{"*"},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: "something",
						},
					},
				},
			}},
		}},
	}

	snap, _ := cache.NewSnapshot("version_wanlin",
		map[resource.Type][]types.Resource{
			resource.EndpointType: {endpointTest},
			resource.ClusterType:  {clusterTest},
			resource.RouteType:    {routeTest},
			resource.ListenerType: {makeEnvoyHTTPListener(), makeGrpcHTTPListener()},
		},
	)
	return snap
}
