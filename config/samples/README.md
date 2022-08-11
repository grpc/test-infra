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

Examples for [PSM benchmarks](../../README.md#psm-benchmarks) can be found
[here](templates/psm/README.md).

[Prometheus](../prometheus/README.md) is enabled by default for all
[PSM benchmarks](../../README.md#psm-benchmarks) tests. Prometheus monitoring
can be enabled to by adding an annotation `enablePrometheus: 'true'` to the load
test configurations. See
[example](config/samples/templates/psm/cxx_example_loadtest_proxied.yaml#l8)
usage of the `enablePrometheus` annotation.

Special considerations is required for running `csharp` examples. These examples
correspond to the legacy C# implementation in [grpc/grpc]. This implementation
was removed by <https://github.com/grpc/grpc/pull/29225>, and is only supported
in version `v1.46.x` and earlier versions of [grpc/grpc]. For the newer C#
implementation in [grpc/grpc-dotnet], see the `dotnet` examples.

[control.proto]:
  https://github.com/grpc/grpc-proto/blob/master/grpc/testing/control.proto
[grpc/grpc]: https://github.com/grpc/grpc
[grpc/grpc-dotnet]: https://github.com/grpc/grpc-dotnet
[grpc/grpc-proto]: https://github.com/grpc/grpc-proto
[grpc oss benchmarks readme]:
  https://github.com/grpc/grpc/blob/master/tools/run_tests/performance/README.md#grpc-oss-benchmarks
