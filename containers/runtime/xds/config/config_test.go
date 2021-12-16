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
	s := ServerResource{
		XDSServerClusterName:   "default_xDSServerClusterName",
		TestServiceClusterName: "default_testServiceClusterName",
		TestRouteName:          "default_TestRouteName",
		TestGrpcListenerName:   "default_testGrpcListenerName",
		TestEnvoyListenerName:  "default_testEnvoyListenerName",
		TestListenerPort:       1234,
		TestUpstreamHost:       "default_testUpstreamHost",
		TestUpstreamPort:       5678,
		TestEndpointName:       "default_testEndpointName",
	}

	currentVersion := "test_version"
	var testTTL time.Duration
	var originalConfig CustomSnapshot
	var processedConfig CustomSnapshot
	var currentResourceType string
	var currentResourceName string

	BeforeEach(func() {
		originalConfig = CustomSnapshot{}
		processedConfig = CustomSnapshot{}
		testTTL, _ = time.ParseDuration("3h")
	})

	It("marshals and unmarshal Endpoint resource correctly", func() {
		currentResourceType = resource.EndpointType
		currentResourceName = s.TestEndpointName
		endpointOnly, err := cache.NewSnapshotWithTTLs(currentVersion, map[resource.Type][]types.ResourceWithTTL{
			currentResourceType: {types.ResourceWithTTL{
				Resource: s.makeEndpoint(),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = CustomSnapshot{endpointOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = CustomSnapshot{}
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
				Resource: s.makeRoute(),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = CustomSnapshot{routeConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = CustomSnapshot{}
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
				Resource: s.makeCluster(),
				TTL:      &testTTL},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = CustomSnapshot{clusterConfigOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = CustomSnapshot{}
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
					Resource: s.makeEnvoyHTTPListener(),
					TTL:      &testTTL,
				},
				types.ResourceWithTTL{
					Resource: s.makeGrpcHTTPListener(),
					TTL:      &testTTL,
				},
			}})
		Expect(err).ToNot(HaveOccurred())

		originalConfig = CustomSnapshot{listenerOnly}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = CustomSnapshot{}
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
		fullSet := s.GenerateSnapshot()

		originalConfig = CustomSnapshot{fullSet}
		marshalConfig, err := json.Marshal(originalConfig)
		Expect(err).ToNot(HaveOccurred())

		processedConfig = CustomSnapshot{}
		err = json.Unmarshal(marshalConfig, &processedConfig)
		Expect(err).ToNot(HaveOccurred())

		// check if the snapshot generate is consistent
		err = processedConfig.Consistent()
		Expect(err).ToNot(HaveOccurred())
	})

	// TODO: Wanlin add test for Secret, Runtime and extension config

})
