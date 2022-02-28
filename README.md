# test-infra - gRPC test infrastructure

This repository contains code for systems that test [gRPC][grpc] which are
versioned, released or deployed separately from the [gRPC Core][grpccore]
codebase.

[grpc]: https://grpc.io
[grpccore]: https://github.com/grpc/grpc

## gRPC OSS benchmarks

gRPC OSS benchmarks are a collection of libraries and executables to schedule,
run and monitor [gRPC performance benchmarking][benchmarking] tests on a
Kubernetes cluster.

The main executable is a [custom controller][] that manages resources of kind
[LoadTest][loadtest]. This controller must be deployed to the cluster before
load tests can be run on it. For deployment information, see [deployment][]. The
controller is implemented with [kubebuilder][].

There is also a set of [tools](tools/README.md) used to generate load test
configurations, prepare prebuilt images and run batches of tests. These tools
are used to run batches of tests for continuous integration.

[Examples](config/samples/README.md) of load test configurations in the
supported languages are also provided.

[custom controller]: cmd/controller/main.go
[deployment]: doc/deployment.md
[kubebuilder]: https://kubebuilder.io
[loadtest]: config/crd/bases/e2etest.grpc.io_loadtests.yaml

## Dashboard

The data generated in continuous integration are saved to [BigQuery][bigquery],
and displayed on a public dashboard linked from the [gRPC performance
benchmarking][benchmarking] page.

For more information, and to build your own dashboard, see
[dashboard](dashboard/README.md).

[bigquery]: https://cloud.google.com/bigquery

## Contributing

Welcome! Please read [how to contribute](CONTRIBUTING.md) before proceeding.

[benchmarking]: https://grpc.io/docs/guides/benchmarking/
