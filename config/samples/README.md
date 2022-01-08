# Examples

This directory contains example configurations for
[LoadTests](../crd/bases/e2etest.grpc.io_loadtests.yaml) in each supported
language.

To see all available options for the LoadTest, please visit the LoadTest
definition in [loadtest_types.go](../../api/v1/loadtest_types.go).

To see all available options for the scenariosJSON embedded in the test, please
visit the definition in [control.proto], in the repository [grpc/grpc-proto].

The examples in this folder are basic examples that build and run the test
components when the test is applied, and do not save data to BigQuery. These can
be run by applying them to the cluster with `kubectl apply -f`.

The examples in the [templates](./templates) folder are templates that use
prebuilt images and require parameter substitution before running.
[Tools](../../tools/README.md) are provided to build images and run tests with
the correct parameters.

All examples are generated from templates stored in the [grpc/grpc] repository.
For more information, please visit the [gRPC OSS benchmarks README] on
[grpc/grpc].

[control.proto]:
  https://github.com/grpc/grpc-proto/blob/master/grpc/testing/control.proto
[grpc/grpc]: https://github.com/grpc/grpc
[grpc/grpc-proto]: https://github.com/grpc/grpc-proto
[grpc oss benchmarks readme]:
  https://github.com/grpc/grpc/blob/master/tools/run_tests/performance/README.md#grpc-oss-benchmarks
