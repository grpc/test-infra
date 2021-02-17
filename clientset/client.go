package v1

import (
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// GRPCTestClientset provides methods to access custom Kubernetes clients for
// testing gRPC.
type GRPCTestClientset interface {
	// LoadTestV1 returns the load test interface, which provides operations on
	// version 1 load tests.
	LoadTestV1() LoadTestInterface
}

type grpcTestV1 struct {
	client rest.Interface
}

func (gv1 *grpcTestV1) LoadTestV1() LoadTestInterface {
	return &loadTestV1{gv1.client}
}

type gRPCTestClient struct {
	client rest.Interface
}

func (gc *gRPCTestClient) LoadTestV1() LoadTestInterface {
	return &loadTestV1{gc.client}
}

// NewForConfig accepts a Kubernetes REST Client config and adds the appropriate
// group, version and serializers to handle requests regarding gRPC test
// resources.
func NewForConfig(c *rest.Config) (GRPCTestClientset, error) {
	config := *c

	gv := grpcv1.GroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(clientgoscheme.Scheme)
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	client, err := rest.UnversionedRESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &gRPCTestClient{client}, nil
}
