package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

type loadTestV1Getter struct {
	ns     string
	client rest.Interface
}

func (l *loadTestV1Getter) Create(test *grpcv1.LoadTest, opts metav1.CreateOptions) (*grpcv1.LoadTest, error) {
	createdTest := &grpcv1.LoadTest{}
	err := l.client.Post().
		Namespace(l.ns).
		Resource("loadtests").
		Body(test).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(createdTest)
	return createdTest, err
}

func (l *loadTestV1Getter) Get(name string, opts metav1.GetOptions) (*grpcv1.LoadTest, error) {
	test := &grpcv1.LoadTest{}
	err := l.client.Get().
		Namespace(l.ns).
		Resource("loadtests").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(test)
	return test, err
}

func (l *loadTestV1Getter) List(opts metav1.ListOptions) (*grpcv1.LoadTestList, error) {
	tests := &grpcv1.LoadTestList{}
	err := l.client.Get().
		Namespace(l.ns).
		Resource("loadtests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(tests)
	return tests, err
}

func (l *loadTestV1Getter) Delete(name string, opts metav1.DeleteOptions) error {
	return l.client.Delete().
		Namespace(l.ns).
		Resource("loadtests").
		Name(name).
		Body(&opts).
		Do().
		Error()
}

type loadTestV1 struct {
	client rest.Interface
}

func (lv *loadTestV1) LoadTests(namespace string) LoadTestGetter {
	return &loadTestV1Getter{
		ns:     namespace,
		client: lv.client,
	}
}
