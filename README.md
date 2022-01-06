# test-infra - gRPC test infrastructure

This repository contains code for systems that test [gRPC] which are versioned,
released or deployed separately from the core [grpc/grpc] codebase.

[grpc]: https://grpc.io
[grpc/grpc]: https://github.com/grpc/grpc

## gRPC OSS benchmarks

gRPC OSS benchmarks are a collection of libraries and executables to schedule,
run and monitor [gRPC performance benchmarking] tests on a Kubernetes cluster.

The main executable is a [custom controller] that manages resources of kind
[LoadTest]. This controller must be deployed to the cluster before load tests
can be run on it. The controller is implemented with [kubebuilder].

There is also a set of tools used to prepare prebuilt images and run batches of
tests. These tools are used to generate the dashboard linked from the [gRPC
performance benchmarking] page. For more information, see
[tools](tools/README.md).

[Examples](config/samples/README.md) of load test configurations in the
supported languages are also provided.

[custom controller]: cmd/controller/main.go
[grpc performance benchmarking]: https://grpc.io/docs/guides/benchmarking/
[kubebuilder]: https://kubebuilder.io
[loadtest]: config/crd/bases/e2etest.grpc.io_loadtests.yaml

## Contributing

Welcome! Please read [how to contribute](CONTRIBUTING.md) before proceeding.

This project includes third party dependencies as git submodules. Be sure to
initialize and update them when setting up a development environment:

```shell
# Init/update during the clone
git clone --recursive https://github.com/grpc/test-infra.git  # HTTPS
git clone --recursive git@github.com:grpc/test-infra.git      # SSH

# (or) Init/update after the clone
git submodule update --init
```
