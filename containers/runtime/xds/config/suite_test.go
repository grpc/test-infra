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

package config

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v3routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	v3httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

type ServerResource struct {
	XDSServerClusterName   string
	TestServiceClusterName string
	TestRouteName          string
	TestGrpcListenerName   string // This field is only used by xDS client
	TestEnvoyListenerName  string
	TestListenerPort       uint // This field is only used by Envoy, socket listener
	TestUpstreamHost       string
	TestUpstreamPort       uint32
	TestEndpointName       string
}

func (s *ServerResource) makeCluster() *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 s.TestServiceClusterName,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
			ServiceName: s.TestEndpointName,
		},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		Http2ProtocolOptions: &core.Http2ProtocolOptions{},
	}
}

func (s *ServerResource) makeEndpoint() *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: s.TestEndpointName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			Locality: &core.Locality{SubZone: "subzone"},
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  s.TestUpstreamHost,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: uint32(s.TestUpstreamPort),
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
}

func (s *ServerResource) makeRoute() *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: s.TestRouteName,
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
							Cluster: s.TestServiceClusterName,
						},
					},
				},
			}},
		}},
	}
}

func (s *ServerResource) makeGrpcHTTPListener() *listener.Listener {
	a, _ := anypb.New(&v3routerpb.Router{})

	hcm, _ := anypb.New(&v3httppb.HttpConnectionManager{
		RouteSpecifier: &v3httppb.HttpConnectionManager_Rds{Rds: &v3httppb.Rds{
			ConfigSource: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{Ads: &core.AggregatedConfigSource{}},
			},
			RouteConfigName: s.TestRouteName,
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
		Name:        s.TestGrpcListenerName,
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

func (s *ServerResource) makeEnvoyHTTPListener() *listener.Listener {
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
				RouteConfigName: s.TestRouteName,
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
		Name: s.TestEnvoyListenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(s.TestListenerPort),
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

// GenerateSnapshot generate the snapshot for both gRPC and Envoy to consume
func (s *ServerResource) GenerateSnapshot() cache.Snapshot {
	snap, _ := cache.NewSnapshot("1",
		map[resource.Type][]types.Resource{
			resource.ClusterType:  {s.makeCluster()},
			resource.RouteType:    {s.makeRoute()},
			resource.ListenerType: {s.makeGrpcHTTPListener(), s.makeEnvoyHTTPListener()},
			resource.EndpointType: {s.makeEndpoint()},
		},
	)
	return snap
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Config Suite",
		[]Reporter{printer.NewlineReporter{}})
}
