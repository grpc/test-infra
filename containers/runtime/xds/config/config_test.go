package config

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/proto"
)

var _ = Describe("config marshal and unmarshal", func() {
	s := TestResource{
		XDSServerClusterName:   "default_xDSServerClusterName",
		TestServiceClusterName: "default_testServiceClusterName",
		TestRouteName:          "default_TestRouteName",
		TestGrpcListenerName:   "default_testGrpcListenerName",
		TestEnvoyListenerName:  "default_testEnvoyListenerName",
		TestListenerPort:       1234,
		TestEndpoints: []*TestEndpoint{{
			TestUpstreamHost: "default_testUpstreamHost",
			TestUpstreamPort: 5678,
		}},
		TestEndpointName: "default_testEndpointName",
	}

	currentVersion := "test_version"
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
		currentResourceName = s.TestEndpointName
		endpointOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {types.ResourceWithTTL{
				Resource: makeEndpoint(s.TestEndpointName, s.TestEndpoints[0].TestUpstreamHost, s.TestEndpoints[0].TestUpstreamPort),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{endpointOnly}
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
		currentResourceName = s.TestRouteName
		routeConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {types.ResourceWithTTL{
				Resource: makeRoute(s.TestRouteName, s.TestServiceClusterName),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{routeConfigOnly}
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
		currentResourceName = s.TestServiceClusterName
		clusterConfigOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {types.ResourceWithTTL{
				Resource: makeCluster(s.TestServiceClusterName, s.TestEndpointName),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{clusterConfigOnly}
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
					Resource: makeEnvoyHTTPListener(s.TestRouteName, s.TestEnvoyListenerName, uint32(s.TestListenerPort)),
					TTL:      &testTTL,
				},
				types.ResourceWithTTL{
					Resource: makeGrpcHTTPListener(s.TestRouteName, s.TestGrpcListenerName, uint32(s.TestListenerPort)),
					TTL:      &testTTL,
				},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = customSnapshot{listenerOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check the version of the resource is processed correctly
		Expect(reflect.DeepEqual(originalConfig.GetVersion(currentResourceType), processedConfig.GetVersion(currentResourceType))).To(BeTrue())

		// gRPC Listeners
		originalResourceGRPC := originalConfig.GetResourcesAndTTL(currentResourceType)[s.TestGrpcListenerName]
		processedResourceGRPC := processedConfig.GetResourcesAndTTL(currentResourceType)[s.TestGrpcListenerName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResourceGRPC.Resource, processedResourceGRPC.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResourceGRPC.TTL, processedResourceGRPC.TTL)).To(BeTrue())

		// Envoy Listeners
		originalResourceEnvoy := originalConfig.GetResourcesAndTTL(currentResourceType)[s.TestGrpcListenerName]
		processedResourceEnvoy := processedConfig.GetResourcesAndTTL(currentResourceType)[s.TestGrpcListenerName]

		// check the resource is processed correctly
		Expect(proto.Equal(originalResourceEnvoy.Resource, processedResourceEnvoy.Resource)).To(BeTrue())

		// check the TTL of the resource is processed correctly
		Expect(reflect.DeepEqual(originalResourceEnvoy.TTL, processedResourceEnvoy.TTL)).To(BeTrue())

	})

	It("marshals and unmarshal all resources correctly", func() {

		fullSet, _ := cache.NewSnapshot("1",
			map[resource.Type][]types.Resource{
				resource.ClusterType:  {makeCluster(s.TestServiceClusterName, s.TestEndpointName)},
				resource.RouteType:    {makeRoute(s.TestRouteName, s.TestServiceClusterName)},
				resource.ListenerType: {makeEnvoyHTTPListener(s.TestRouteName, s.TestEnvoyListenerName, uint32(s.TestListenerPort)), makeGrpcHTTPListener(s.TestRouteName, s.TestGrpcListenerName, uint32(s.TestListenerPort))},
				resource.EndpointType: {makeEndpoint(s.TestEndpointName, s.TestEndpoints[0].TestUpstreamHost, s.TestEndpoints[0].TestUpstreamPort)},
			})

		originalConfig = customSnapshot{fullSet}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = customSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check if the snapshot generate is consistent
		err = processedConfig.Consistent()
		Expect(err).ToNot(HaveOccurred())
	})
	// TODO: Wanlin add test for Secret, Runtime and extension config
})
