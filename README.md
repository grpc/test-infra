# test-infra

(Soon-to-become-official) Repo for gRPC testing infrastructure support code

The test-infra repository contains code for systems that test gRPC which are
versioned, released or deployed separately from the [grpc/grpc] codebase.

[grpc/grpc]: https://github.com/grpc/grpc

## [Benchmarks]

gRPC Benchmarks is a collection of libraries and executables to schedule, run
and monitor gRPC benchmarks on a Kubernetes cluster.

## Contribute

Welcome! Please read and follow the steps in the
[CONTRIBUTING.md](CONTRIBUTING.md) file.

This project includes third party dependencies as git submodules. Be sure to
initialize and update them when setting up a development environment:

```shell
# Init/update during the clone
git clone --recursive https://github.com/grpc/test-infra.git  # HTTPS
git clone --recursive git@github.com:grpc/test-infra.git      # SSH

# (or) Init/update after the clone
git submodule update --init
```
