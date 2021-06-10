# Ready

Ready is a container that waits for a list of pods within a load test to
become available. It exits successfully when all worker pods are ready, writing a
comma-separated list of their IP addresses to a file. It exits unsuccessfully if
a timeout was exceeded before all pods were ready.

## Usage

The container relies on command line argument to specify the load test's name.
For example,

```shell
go run ready.go LOADTEST_NAME
```

Will wait for all worker pods which belongs to LOADTEST_NAME

Meanwhile, users can set environment variables to override some defaults:

- `$READY_TIMEOUT` specifies the maximum amount of time the container should
  wait for pods to become ready. This time is specified in a human-readable
  format that is parsable by Go's
  [time.ParseDuration](https://pkg.go.dev/time?tab=doc#ParseDuration). For
  example, "1m" represents 1 minute and "3h" represents 3 hours. If this
  timeout is reached before the pods are ready, the container will exit with a
  code of 1.

- `$READY_OUTPUT_FILE` specifies the absolute path of the output file. This
  will contain a comma-separated list of IP addresses for matching pods. This
  defaults to /tmp/loadtest_workers.

- `$KUBE_CONFIG` specifies the path to a Kubernetes config file. This can be
  omitted when running in a Kubernetes cluster. If running outside a cluster,
  this is required. It will likely be ~/.kube/config when developing locally
  on Linux.

## Building

This image requires some utility code outside of this directory. Therefore, the
test-infra/ directory should be used as the build context:

```shell
cd ../../../  # should be test-infra/
docker build -f containers/init/ready/Dockerfile .
```
