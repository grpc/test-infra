# Examples

This directory contains a [LoadTest](e2etest.grpc.io_v1_loadtest.yaml) and an
example [scenario](java_example_scenario.json) for Java.

The LoadTest references a ConfigMap named *java-example-scenario*, which should
contain the contents of the scenario protobuf as JSON. Once you have
authenticated with a Kubernetes cluster, this command will create it:

```
$ kubectl create configmap java-example-scenario \
    --from-file=java_example_scenario.json
```

Every ConfigMap must have a unique name. You will need to choose a different
name if this one already exists.

Notice that the ConfigMap's name uses dashes instead of underscores. gRPC uses
underscores by convention, but Kubernetes requires valid DNS names for all
resources. For this reason, we use dashes when discussing the Kubernetes
ConfigMap and underscores when discussing the proto itself.

To see all available options for the Scenario proto, please visit its definition
in grpc/grpc-proto:
https://github.com/grpc/grpc-proto/blob/master/grpc/testing/control.proto.

To see all available options for the LoadTest, please visit its definition in
grpc/test-infra's api/v1 directory:
[loadtest_types.go](../../api/v1/loadtest_types.go).

After testing, you can delete the ConfigMap using:

```
$ kubectl delete configmap java-example-scenario
```
