# Tools

The tools contained in this folder enable a user to run multiple tests, wait for
them to finish, and generate an xml report with pass/fail results.

These tools are intended to work with load test configurations generated from
load test scenarios by tools stored in repository [grpc/grpc]. For information
on those tools and examples of the tools being used together, see the
[gRPC OSS benchmarks README](https://github.com/grpc/grpc/blob/master/tools/run_tests/performance/README.md#grpc-oss-benchmarks)
in that repository.

[grpc/grpc]: https://github.com/grpc/grpc

## Building the tools

You can also run any of the tools in this folder with
`go run tools/cmd/${tool}/main.go`.

You can also build tool binaries using the makefile:

```shell
make all-tools
```

You can then invoke the binary for each tool as `bin/${tool}`.

## Test runner

The [runner](cmd/runner/main.go) tool runs collections of tests, optionally
assigning them to separate _queues_. The queue name for each test is taken from
an annotation in the test configuration. The key for this annotation is
specified by the option `annotation-key`.

The runner applies tests to the cluster according to the concurrency level for
each queue, polls the tests while they are running and collects results to
compose a report.

The input files for the runner are multi-part yaml files containing load test
configurations. The (optional) output is an xml report in xunit format.

The `runner` tool takes the following options:

- `-annotation-key`<br> annotation key to parse for queue assignment (default:
  `pool`).
- `-c`<br> Concurrency level, in the form `[<queue name>:]<concurrency level>`.
- `-i`<br> Input files containing load test configurations.
- `-o`<br> Name of the output file for xunit xml report.
- `-polling-interval`<br> polling interval for load test status (default:
  `20s`).
- `-polling-retries`<br> Maximum retries in case of communication failure
  (default: `2`).
- `-xunit-suites-name`<br> Name field for testsuites in xunit xml report.

The following example runs tests on two separate queues, specified by the `pool`
annotation (the most common case in production, where tests run simultaneously
on separate node pools):

```shell
bin/runner -i input.yaml -c queue1:2 -c queue2:3 -o sponge_log.xml
```

The following examples runs tests from two different files on a single queue
(useful for tests that run on a default pool):

```shell
bin/runner -i input1.yaml input2.yaml -annotation_key= -c :2
```

The queue in the second example is unnamed. The two examples represent choices
in queue assignment: If a named queue is specified, then all queues must be
named and assigned a concurrency level; If an unnamed queue is specified, then
it must be the only queue and all tests must be assigned to it.

## Using prebuilt images with gRPC OSS benchmarks

The tools [prepare_prebuilt_workers](cmd/prepare_prebuilt_workers/main.go) and
[delete_prebuilt_workers](cmd/delete_prebuiilt_workers/main.go):

- Build the images with the worker executables compiled and embedded.
- Push the images to specified registry.
- Delete the images from specified registry.

### Build and push images

The [prepare_prebuilt_workers](cmd/prepare_prebuilt_workers/main.go) tool builds
images and pushes them to a user specified Google cloud registry. For example,
the following shows this process (building and pushing prebuilt images) for
`cxx` and `go` workers:

```shell
bin/prepare_prebuilt_workers \
     -l cxx:master \
     -l go:master \
     -p "${image_registry}" \
     -t "${tag}" \
     -r containers/pre_built_workers
```

This builds `cxx` and `go` images contain workers built from the commit/branch
we wish to test.

The tool `prepare_prebuilt_workers` takes the following options:

- `-l`<br> Language and GITREF to benchmark. The language and its specific
  GITREF wish to build workers from can be specified as `language:COMMIT_SHA`.
  May be repeated. Valid input for language names are all in lower case:
  `csharp`, `c++`/`cxx`, `go`, `java`, `node`, `node_purejs`, `php7`.
  `php7_protobuf_c`, `python`, `python_asyncio` and `ruby`.
- `-t`<br> Tag for prebuilt images. Tag is a required field. Tag complies with
  [docker tag's restrictions](https://docs.docker.com/engine/reference/commandline/tag/#extended-description).
- `-r`<br> Root directory of Dockerfiles.
- `-build-only`<br> Option to build only, when specified `-build-only=true`
  images will only be built locally.
- `-p`<br> Image registry to store images. The image prefix should be in form of
  `gcr.io/project-name/your-directory-name` for the Google Container Registry.
  For example, the built images would be stored as
  `gcr.io/grpc-testing/project-name/pre_built_workers/cxx:user-specified-tag`
  and `gcr.io/grpc-testing/project-name/pre_built_workers/go:user-specified-tag`
  .

If using a registry other than GCR, the images should be built through the
script with the flag `-build-only=true`. The user could then push the images
manually.

The Dockerfiles that the script uses to build are available in
[../containers/pre_built_workers](../containers/pre_built_workers).

### Delete the images

The tool [delete_prebuilt_workers](cmd/delete_prebuilt_workers/main.go) deletes
images within a user specified registry. The script lists all images within the
specified registry, then checks if the image has the user specified tag. This
script only supports Google Container Registry, because it relies on the
google-cloud-sdk.

The following example deletes all images within `${image_registry}` that have
tag `${tag}`:

```shell
bin/delete_prebuilt_workers \
    -p "${image_registry}" \
    -t "${tag}"
```

- `-t`<br> Tag for prebuilt images. Tag is the unique identifier for images to
  delete. If the image has multiple tags including the user specified tag, the
  tag specified here is removed from image's tag list, otherwise the image is
  deleted.
- `-p`<br> Image registry to search images from. Only accepts the image registry
  prefix, not the actual image name. If the image registry supports nested
  repositories, the image registry prefix should be the absolute path to the
  image's parent repository.
