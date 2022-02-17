# Deployment

This page explains how to set up a testbed to run [gRPC OSS benchmarks]. The
testbed consists of a Kubernetes cluster and a custom controller deployed to the
cluster. For other aspects of running tests once the testbed is set up, see
[tools].

[grpc oss benchmarks]: ../README.md#grpc-oss-benchmarks
[tools]: ../tools/README.md

## Cluster setup

The cluster running benchmark jobs must be configured with node pools
dimensioned for the number of simultaneous tests that it should support. The
controller uses `pool` as a node selector for the various pod types. Worker pods
have mutual anti-affinity, so one node is required per pod.

For example, the node pools that are used in our continuous integration testbed
are configured as follows:

| Pool name            | Node count | Machine type   | Kubernetes labels                          |
| :------------------- | ---------: | :------------- | :----------------------------------------- |
| system               |          2 | e2-standard-8  | default-system-pool:true,&nbsp;pool:system |
| drivers-ci           |          8 | e2-standard-2  | pool:drivers-ci                            |
| workers-c2-8core-ci  |          8 | c2-standard-8  | pool:workers-c2-8core-ci                   |
| workers-c2-30core-ci |          8 | c2-standard-30 | pool:workers-c2-30core-ci                  |

Since each scenario in our tests requires one driver and two workers, this
configuration supports four simultaneous tests on 8-core machines and four on
30-core machines. Drivers require few resources, and do not have mutual
anti-affinity. We find it convenient to schedule them on two-core machines with
a node count set to the required number of drivers, rather than on a larger
shared machine, because that allows the driver pool to be resized together with
the worker pools. The controller itself is scheduled in the `system` pool.

In addition to the pools used in continuous integration, our cluster contains
another set of node pools that can be used for ad hoc testing:

| Pool name      | Node count | Machine type   | Kubernetes labels                                 |
| :------------- | ---------: | :------------- | :------------------------------------------------ |
| drivers        |          8 | e2-standard-8  | default-driver-pool:true,&nbsp;pool:drivers       |
| workers-8core  |          8 | e2-standard-8  | default-worker-pool:true,&nbsp;pool:workers-8core |
| workers-32core |          8 | e2-standard-32 | pool:workers-32core                               |

Some pools are labeled with `default-*-pool` labels. These labels specify which
pool to use if it is not specified in the LoadTest configuration. With the
configuration above, these tests (for instance, the tests specified in the
[examples]) will use the `drivers` and `workers-8core` pools, and not interfere
with continuous integration jobs. The default labels are defined as part of the
[controller configuration](#controller-configuration): if they are not set, the
controller will only run tests where the `pool` labels are specified explicitly.

## Controller setup

The following instructions explain how to build the custom LoadTest controller
and how to deploy it to the cluster.

### Cloning the repo

In order to build a specific version of the controller, you must check out the
desired version. The following commands clone the repo and check out version
`v1.0.0`:

```shell
git clone https://github.com/grpc/test-infra && cd test-infra
git checkout tags/v1.0.0
```

### Environment variables

The following environment variables must be set before starting the build:

- `TEST_INFRA_VERSION`
- `DRIVER_VERSION`
- `INIT_IMAGE_PREFIX`
- `BUILD_IMAGE_PREFIX`
- `RUN_IMAGE_PREFIX`
- `KILL_AFTER`

`TEST_INFRA_VERSION` is used to tag the images created by the controller build,
and defaults to `latest`. The other variables must be set explicitly.

`DRIVER_VERSION` is the version of the load test driver. The driver is built
from the [gRPC Core] repository, so it should be set to the latest release of
gRPC (e.g. `v1.44.0`).

`INIT_IMAGE_PREFIX`, `BUILD_IMAGE_PREFIX` and `RUN_IMAGE_PREFIX` define the
repository locations where various kinds of images will be uploaded.

`KILL_AFTER` is the time interval in seconds after which a KILL signal will be
sent to test components, if they have not terminated after timeout. Component
timeout is set in the LoadTest configuration. `KILL_AFTER` is set in the
[controller configuration], as a safeguard for components that may hang and
consume resources after test timeout.

The variables used to build the `v1.0.0` release are as follows:

```shell
export TEST_INFRA_VERSION=v1.0.0
export DRIVER_VERSION=v1.44.0
export INIT_IMAGE_PREFIX=gcr.io/grpc-testing/e2etest/init/
export BUILD_IMAGE_PREFIX=gcr.io/grpc-testing/e2etest/init/build/
export RUN_IMAGE_PREFIX=gcr.io/grpc-testing/e2etest/init/runtime/
export KILL_AFTER=30
```

Our images are pushed to `gcr.io`. You can push to any image repository by
changing the environment variables.

You can change `TEST_INFRA_VERSION` to any label you would like to apply to your
images.

### Controller configuration

The controller requires a configuration file to be included in the controller
image. This configuration file can be generated from a template as follows:

```shell
go run config/cmd/configure.go \
    -version="${TEST_INFRA_VERSION}" \
    -init-image-prefix="${INIT_IMAGE_PREFIX}" \
    -build-image-prefix="${BUILD_IMAGE_PREFIX}" \
    -run-image-prefix="${RUN_IMAGE_PREFIX}" \
    -kill-after="${KILL_AFTER}" \
    -validate=true \
    config/defaults_template.yaml \
    config/defaults.yaml
```

This step must be completed before
[building and pushing images](#building-and-pushing-images).

The structure of the configuration file can be seen in [defaults_template.yaml].

The controller configuration contains default pool labels (see
[cluster setup](#cluster-setup)) and the value of `KILL_AFTER`, in addition to
the location of images generated when
[building and pushing images](#building-and-pushing-images).

[defaults_template.yaml]: ../config/defaults_template.yaml

### Building and testing

The controller binary can be built and tested with the following command:

```shell
make all test
```

### Building and pushing images

Images can be built and pushed to an image repository with the following
command:

```shell
make all-images push-all-images
```

The set of images includes the controller, the driver runtime, and a ready
container image used by the driver. These images must be included in any build.

The set of images also includes build and runtime images for every supported
language, plus a language-agnostic clone container image. These images are
necessary to run any tests that do not use [pre-built images], such as the
[examples].

The complete set of images built for `v1.0.0` is as follows:

```shell
gcr.io/grpc-testing/e2etest/init/build/csharp:v1.0.0
gcr.io/grpc-testing/e2etest/init/build/node:v1.0.0
gcr.io/grpc-testing/e2etest/init/build/php7:v1.0.0
gcr.io/grpc-testing/e2etest/init/build/ruby:v1.0.0
gcr.io/grpc-testing/e2etest/init/clone:v1.0.0
gcr.io/grpc-testing/e2etest/init/ready:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/controller:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/cxx:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/driver:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/go:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/java:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/node:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/php7:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/python:v1.0.0
gcr.io/grpc-testing/e2etest/runtime/ruby:v1.0.0
```

This should match what is included in the [controller configuration].

[pre-built images]:
  ../tools/README.md#using-prebuilt-images-with-grpc-oss-benchmarks

## Deleting the previous deployment

The following command deletes the previous deployment from the cluster:

```shell
kubectl -n test-infra-system delete deployments --all
```

This is an optional step, but may be advisable, so we can start from a clean
deployment.

## Deploying to the cluster

Assuming that you are connected to the cluster where you want to deploy, the
controller can be deployed as follows:

```shell
make deploy install
```

This step depends only on `TEST_INFRA_VERSION` and `RUN_IMAGE_PREFIX`.

The command above can also be used to deploy an existing version of the
controller. In this case, the environment variables should point to the location
of the controller binary.

## Verifying the deployment

You can verify that the deployment started by running the following command:

```shell
kubectl -n test-infra-system get deployments
```

You should eventually see `1/1` for the `READY` column, in the command output:

```shell
NAME                   READY   UP-TO-DATE   AVAILABLE   AGE
controller-manager      1/1        1            1           18s
```

Verify that the deployment is running in the `system` node pool by running the
following command:

```shell
kubectl get pods -n test-infra-system \
  -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName
```

The node names in the response should contain the word `system`:

```shell
kubectl get pods -n test-infra-system \
  -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName
```

It may take a while for the deployment to start. If it does not, you will need
to debug the deployment by checking the description of its pod and the logs of
its `manager` container. The deployment runs in namespace `test-infra-system`.

## Running an example test

Verify that the deployment is able to run a test by running the example Go test:

1. Start the test:

   ```shell
   kubectl apply -f config/samples/go_example_loadtest.yaml
   ```

1. Check the status of the test:

   ```shell
   kubectl get loadtest -l prefix=examples,language=go -o jsonpath='{range .items[*]}{.status.state}{"\n"}{end}'
   ```

   Initially, the status should show `Running`.

1. Repeat the previous step until the status changes to `Succeeded`.

1. Delete the test:

   ```shell
   kubectl delete loadtest -l prefix=examples,language=go
   ```

[examples]: ../config/samples/README.md
[grpc core]: https://github.com/grpc/grpc
