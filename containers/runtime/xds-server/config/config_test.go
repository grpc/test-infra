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
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	testres "github.com/envoyproxy/go-control-plane/pkg/test/resource/v3"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("config marshal and unmarshal", func() {
	currentVersion := "testVersion"
	testGrpcListenerName := "defaultTestGrpcListenerName"
	testEnvoyListenerPort := 1234
	endpoints := []*TestEndpoint{{
		TestUpstreamHost: "defaultTestUpstreamHost",
		TestUpstreamPort: 5678,
	}}
	testServiceClusterName := "defaultTestServiceClusterName"
	testEnvoyListenerName := "defaultTestEnvoyListenerName"
	testRouteName := "defaultTestRouteName"
	testEndpointName := "defaultTestEndpointName"
	var testTTL time.Duration
	var originalConfig customSnapshot
	var processedConfig customSnapshot
	var currentResourceType string
	var currentResourceName string

	BeforeEach(func() {
		originalConfig = customSnapshot{}
		processedConfig = customSnapshot{}
		testTTL, _ = time.ParseDuration("3h")
	})

	It("marshals and unmarshal Endpoint resource correctly", func() {
		currentResourceType = resource.EndpointType
		currentResourceName = testEndpointName
		endpointOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {types.ResourceWithTTL{
				Resource: makeEndpoint(testEndpointName, endpoints[0].TestUpstreamHost, endpoints[0].TestUpstreamPort),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*endpointOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		originalResource := originalConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]
		processedResource := processedConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResource.Resource, processedResource.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResource.TTL, processedResource.TTL)).To(BeTrue())
	})

	It("marshals and unmarshal RouteConfiguration resource correctly", func() {
		currentResourceType = resource.RouteType
		currentResourceName = testRouteName
		routeConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {types.ResourceWithTTL{
				Resource: makeRoute(testRouteName, testServiceClusterName),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*routeConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		originalResource := originalConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]
		processedResource := processedConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResource.Resource, processedResource.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResource.TTL, processedResource.TTL)).To(BeTrue())
	})

	It("marshals and unmarshal Cluster resource correctly", func() {
		currentResourceType = resource.ClusterType
		currentResourceName = testServiceClusterName
		clusterConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {types.ResourceWithTTL{
				Resource: makeCluster(testServiceClusterName, testEndpointName),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*clusterConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		originalResource := originalConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]
		processedResource := processedConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResource.Resource, processedResource.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResource.TTL, processedResource.TTL)).To(BeTrue())
	})

	It("marshals and unmarshal Listener resource correctly", func() {
		currentResourceType = resource.ListenerType
		listenerOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {
				types.ResourceWithTTL{
					Resource: makeEnvoyHTTPListener(testRouteName, testEnvoyListenerName, uint32(testEnvoyListenerPort)),
					TTL:      &testTTL,
				},
				types.ResourceWithTTL{
					Resource: makeGrpcHTTPListener(testRouteName, testGrpcListenerName),
					TTL:      &testTTL,
				},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*listenerOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		// gRPC Listeners
		originalResourceGRPC := originalConfig.GetResourcesAndTTL(currentResourceType)[testGrpcListenerName]
		processedResourceGRPC := processedConfig.GetResourcesAndTTL(currentResourceType)[testGrpcListenerName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResourceGRPC.Resource, processedResourceGRPC.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResourceGRPC.TTL, processedResourceGRPC.TTL)).To(BeTrue())

		// Envoy Listeners
		originalResourceEnvoy := originalConfig.GetResourcesAndTTL(currentResourceType)[testGrpcListenerName]
		processedResourceEnvoy := processedConfig.GetResourcesAndTTL(currentResourceType)[testGrpcListenerName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResourceEnvoy.Resource, processedResourceEnvoy.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResourceEnvoy.TTL, processedResourceEnvoy.TTL)).To(BeTrue())

	})

	It("marshals and unmarshal Runtime resource correctly", func() {
		currentResourceType = resource.RuntimeType
		runtimeConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {
				types.ResourceWithTTL{
					Resource: testres.MakeRuntime("runtimeName "),
					TTL:      &testTTL,
				},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*runtimeConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		originalResource := originalConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]
		processedResource := processedConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResource.Resource, processedResource.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResource.TTL, processedResource.TTL)).To(BeTrue())
	})

	It("marshals and unmarshal Secret resource correctly", func() {
		currentResourceType = resource.SecretType

		var secrets []types.ResourceWithTTL
		for _, se := range testres.MakeSecrets("tlsName", "rootName") {
			secrets = append(secrets, types.ResourceWithTTL{
				Resource: se,
				TTL:      &testTTL,
			})
		}

		secretConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: secrets,
		})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*secretConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		originalResource := originalConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]
		processedResource := processedConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResource.Resource, processedResource.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResource.TTL, processedResource.TTL)).To(BeTrue())
	})

	It("marshals and unmarshal ExtensionConfig resource correctly", func() {
		currentResourceType = resource.ExtensionConfigType
		extensionConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {
				types.ResourceWithTTL{
					Resource: testres.MakeExtensionConfig("ads", "extensionConfigName", testRouteName),
					TTL:      &testTTL,
				},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*extensionConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		originalResource := originalConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]
		processedResource := processedConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResource.Resource, processedResource.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResource.TTL, processedResource.TTL)).To(BeTrue())
	})

	It("marshals and unmarshal ScopedRoute resource correctly", func() {
		currentResourceType = resource.ScopedRouteType
		scopedRouteConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {
				types.ResourceWithTTL{
					Resource: testres.MakeScopedRouteConfig("scopedRouteName", testRouteName, []string{"1.2.3.4"}),
					TTL:      &testTTL,
				},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{*scopedRouteConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		originalResource := originalConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]
		processedResource := processedConfig.GetResourcesAndTTL(currentResourceType)[currentResourceName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResource.Resource, processedResource.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResource.TTL, processedResource.TTL)).To(BeTrue())
	})

	It("marshals and unmarshal multiple resources correctly", func() {

		fullSet, _ := cache.NewSnapshot(currentVersion,
			map[resource.Type][]types.Resource{
				resource.ClusterType:  {makeCluster(testServiceClusterName, testEndpointName)},
				resource.RouteType:    {makeRoute(testRouteName, testServiceClusterName)},
				resource.ListenerType: {makeEnvoyHTTPListener(testRouteName, testEnvoyListenerName, uint32(testEnvoyListenerPort)), makeGrpcHTTPListener(testRouteName, testGrpcListenerName)},
				resource.EndpointType: {makeEndpoint(testEndpointName, endpoints[0].TestUpstreamHost, endpoints[0].TestUpstreamPort)},
			})

		originalConfig = customSnapshot{*fullSet}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check if the snapshot generate is consistent
		err = processedConfig.Consistent()
		Expect(err).ToNot(HaveOccurred())
	})
})
var _ = Describe("Update Endpoint", func() {

	var snap *cache.Snapshot
	var endpoints []TestEndpoint

	currentVersion := "testVersion"
	testServiceClusterName := "defaultTestServiceClusterName"
	testEnvoyListenerName := "defaultTestEnvoyListenerName"
	testRouteName := "defaultTestRouteName"
	testEndpointName := "defaultTestEndpointName"
	testGrpcListenerName := "defaultTestGrpcListenerName"
	testEnvoyListenerPort := 1234
	originalEndpoints := []TestEndpoint{{
		TestUpstreamHost: "defaultTestUpstreamHost",
		TestUpstreamPort: 5678,
	}}

	BeforeEach(func() {
		snap, _ = cache.NewSnapshot(currentVersion,
			map[resource.Type][]types.Resource{
				resource.ClusterType:  {makeCluster(testServiceClusterName, testEndpointName)},
				resource.RouteType:    {makeRoute(testRouteName, testServiceClusterName)},
				resource.ListenerType: {makeEnvoyHTTPListener(testRouteName, testEnvoyListenerName, uint32(testEnvoyListenerPort)), makeGrpcHTTPListener(testRouteName, testGrpcListenerName)},
				resource.EndpointType: {makeEndpoint(testEndpointName, originalEndpoints[0].TestUpstreamHost, originalEndpoints[0].TestUpstreamPort)},
			})
	})
	It("returns err when the number of endpoints doesn't match", func() {
		endpoints = []TestEndpoint{{
			TestUpstreamHost: "test-host-1",
			TestUpstreamPort: 1,
		}, {
			TestUpstreamHost: "test-host-2",
			TestUpstreamPort: 2,
		}}

		err := UpdateEndpoint(snap, endpoints)

		Expect(err).To(HaveOccurred())
	})
	It("update the endpoints", func() {
		endpoints = []TestEndpoint{{
			TestUpstreamHost: "test-host-1",
			TestUpstreamPort: uint32(1),
		}}

		err := UpdateEndpoint(snap, endpoints)

		endpointResource := snap.Resources[int(cache.GetResponseType(resource.EndpointType))].Items[testEndpointName].Resource
		endpointData, err := protojson.Marshal(endpointResource)

		endpointService := endpoint.ClusterLoadAssignment{}
		err = protojson.Unmarshal(endpointData, &endpointService)

		Expect(err).ToNot(HaveOccurred())
		Expect(endpointService.Endpoints[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().Address).To(Equal("test-host-1"))
		Expect(endpointService.Endpoints[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().GetPortValue()).To(Equal(uint32(1)))
	})

})

var _ = Describe("SocketListenerOnly", func() {
	var snap *cache.Snapshot

	currentVersion := "testVersion"
	testServiceClusterName := "defaultTestServiceClusterName"
	testEnvoyListenerName := "defaultTestEnvoyListenerName"
	testRouteName := "defaultTestRouteName"
	testEndpointName := "defaultTestEndpointName"
	testGrpcListenerName := "defaultTestGrpcListenerName"
	testEnvoyListenerPort := 1234
	endpoints := []*TestEndpoint{{
		TestUpstreamHost: "defaultTestUpstreamHost",
		TestUpstreamPort: 5678,
	}}

	BeforeEach(func() {
		snap, _ = cache.NewSnapshot(currentVersion,
			map[resource.Type][]types.Resource{
				resource.ClusterType:  {makeCluster(testServiceClusterName, testEndpointName)},
				resource.RouteType:    {makeRoute(testRouteName, testServiceClusterName)},
				resource.ListenerType: {makeEnvoyHTTPListener(testRouteName, testEnvoyListenerName, uint32(testEnvoyListenerPort)), makeGrpcHTTPListener(testRouteName, testGrpcListenerName)},
				resource.EndpointType: {makeEndpoint(testEndpointName, endpoints[0].TestUpstreamHost, endpoints[0].TestUpstreamPort)},
			})
	})
	It("leaves only the socket listeners", func() {
		err := IncludeSocketListenerOnly(snap)
		Expect(err).ToNot(HaveOccurred())

		_, grpcListenerExist := snap.Resources[int(cache.GetResponseType(resource.ListenerType))].Items[testGrpcListenerName]
		Expect(grpcListenerExist).To(BeFalse())

		_, envoyListenerExist := snap.Resources[int(cache.GetResponseType(resource.ListenerType))].Items[testEnvoyListenerName]
		Expect(envoyListenerExist).To(BeTrue())
	})

})

var _ = Describe("ConstructProxylessTestTarget", func() {
	var snap *cache.Snapshot

	currentVersion := "testVersion"
	testServiceClusterName := "defaultTestServiceClusterName"
	testEnvoyListenerName := "defaultTestEnvoyListenerName"
	testRouteName := "defaultTestRouteName"
	testEndpointName := "defaultTestEndpointName"
	testGrpcListenerName := "defaultTestGrpcListenerName"
	testEnvoyListenerPort := 1234
	endpoints := []TestEndpoint{{
		TestUpstreamHost: "defaultTestUpstreamHost",
		TestUpstreamPort: 5678,
	}}

	BeforeEach(func() {
		snap, _ = cache.NewSnapshot(currentVersion,
			map[resource.Type][]types.Resource{
				resource.ClusterType:  {makeCluster(testServiceClusterName, testEndpointName)},
				resource.RouteType:    {makeRoute(testRouteName, testServiceClusterName)},
				resource.ListenerType: {makeEnvoyHTTPListener(testRouteName, testEnvoyListenerName, uint32(testEnvoyListenerPort)), makeGrpcHTTPListener(testRouteName, testGrpcListenerName)},
				resource.EndpointType: {makeEndpoint(testEndpointName, endpoints[0].TestUpstreamHost, endpoints[0].TestUpstreamPort)},
			})
	})
	It("finds the proxyless test server target ", func() {
		target, err := ConstructProxylessTestTarget(snap)

		Expect(err).ToNot(HaveOccurred())

		expected := "xds:///" + testGrpcListenerName

		Expect(expected == target).To(BeTrue())
	})
})

var _ = Describe("ConstructProxiedTestTarget", func() {
	var snap *cache.Snapshot

	currentVersion := "testVersion"
	testServiceClusterName := "defaultTestServiceClusterName"
	testEnvoyListenerName := "defaultTestEnvoyListenerName"
	testRouteName := "defaultTestRouteName"
	testEndpointName := "defaultTestEndpointName"
	testGrpcListenerName := "defaultTestGrpcListenerName"
	testEnvoyListenerPort := 1234
	endpoints := []TestEndpoint{{
		TestUpstreamHost: "defaultTestUpstreamHost",
		TestUpstreamPort: 5678,
	}}

	BeforeEach(func() {
		snap, _ = cache.NewSnapshot(currentVersion,
			map[resource.Type][]types.Resource{
				resource.ClusterType:  {makeCluster(testServiceClusterName, testEndpointName)},
				resource.RouteType:    {makeRoute(testRouteName, testServiceClusterName)},
				resource.ListenerType: {makeEnvoyHTTPListener(testRouteName, testEnvoyListenerName, uint32(testEnvoyListenerPort)), makeGrpcHTTPListener(testRouteName, testGrpcListenerName)},
				resource.EndpointType: {makeEndpoint(testEndpointName, endpoints[0].TestUpstreamHost, endpoints[0].TestUpstreamPort)},
			})
	})
	It("finds the proxied test server target ", func() {
		target, err := ConstructProxiedTestTarget(snap)
		Expect(err).ToNot(HaveOccurred())

		expected := "localhost:" + fmt.Sprint(testEnvoyListenerPort)
		Expect(expected == target).To(BeTrue())
	})
})
