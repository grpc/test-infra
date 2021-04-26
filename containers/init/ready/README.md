# Ready

Ready is a container that waits for a list of pods with specific labels to
become available. It exits successfully when all pods are ready, writing a
comma-separated list of their IP addresses to a file. It exits unsuccessfully if
a timeout was exceeded before all pods were ready.

## Usage

The container relies on command line arguments to specify the labels pods should
match. These arguments are in the form of `key=value,key2=value2` where commas
are treated as `AND`s and spaces delineate separate pods. For example,

```shell
go run ready.go role=server,proxy=envoy role=client role=client
```

Will wait for three pods:

1. pod with labels role=server AND proxy=envoy
2. pod with label role=client
3. another pod with label role=client

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
