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

package v1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

type loadTestV1Getter struct {
	ns     string
	client rest.Interface
}

var _ LoadTestGetter = &loadTestV1Getter{}

func (l *loadTestV1Getter) Create(ctx context.Context, test *grpcv1.LoadTest, opts metav1.CreateOptions) (*grpcv1.LoadTest, error) {
	createdTest := &grpcv1.LoadTest{}
	err := l.client.Post().
		Namespace(l.ns).
		Resource("loadtests").
		Body(test).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(createdTest)
	return createdTest, err
}

func (l *loadTestV1Getter) Get(ctx context.Context, name string, opts metav1.GetOptions) (*grpcv1.LoadTest, error) {
	test := &grpcv1.LoadTest{}
	err := l.client.Get().
		Namespace(l.ns).
		Resource("loadtests").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(test)
	return test, err
}

func (l *loadTestV1Getter) List(ctx context.Context, opts metav1.ListOptions) (*grpcv1.LoadTestList, error) {
	tests := &grpcv1.LoadTestList{}
	err := l.client.Get().
		Namespace(l.ns).
		Resource("loadtests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(tests)
	return tests, err
}

func (l *loadTestV1Getter) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return l.client.Delete().
		Namespace(l.ns).
		Resource("loadtests").
		Name(name).
		Body(&opts).
		Do(ctx).
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
