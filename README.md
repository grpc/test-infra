# test-infra - gRPC test infrastructure

This repository contains code for systems that test [gRPC][grpc] which are
versioned, released or deployed separately from the [gRPC Core][grpccore]
codebase.

For an overview, see
[blog post](https://grpc.io/blog/performance-benchmarks-gke/).

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

## PSM benchmarks

This repository now includes infrastructure to support
[service mesh](https://istio.io/latest/about/service-mesh/) benchmarks comparing
dataplane performance of proxyless gRPC service mesh (PSM) deployments and that
of proxied deployments using an Envoy sidecar.

The client pod in PSM benchmarks includes a
[fake xDS server](containers/runtime/xds-server/README.md) that serves as a gRPC
control plane. The client pod in the proxied case also includes an Envoy
[sidecar](containers/runtime/sidecar/).

[Prometheus](config/prometheus/README.md) is used to monitor CPU and memory
utilization in PSM benchmarks.

[Examples](config/samples/templates/psm/README.md) of proxied and proxyless
tests are now available.

This is only an initial release. Additional features and more detailed
documentation will be added in a future release.

## Contributing

Welcome! Please read [how to contribute](CONTRIBUTING.md) before proceeding.

[benchmarking]: https://grpc.io/docs/guides/benchmarking/
