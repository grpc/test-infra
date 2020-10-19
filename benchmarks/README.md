# gRPC Benchmarks

**THIS IS A PROTOTYPE, SEE [THIS GUIDE][benchmark guide] TO LEARN ABOUT THE
SYSTEM THAT IS CURRENTLY IN-USE.**

[benchmark guide]: https://grpc.io/docs/guides/benchmarking/

gRPC benchmarks is a collection of libraries and executables to schedule, run
and monitor gRPC benchmarks on a Kubernetes cluster.

## Purpose

1.  Collect reliable data through a hermetic environment
2.  Improve developer velocity by providing high availability
3.  Provide a transparent environment that external contributors can replicate
4.  Make experimentation as painless as possible
5.  Minimize manual maintenance and administrative tasks

## Dependencies

- [Go 1.14](https://golang.org)
- [Docker](https://docker.com)
- [GKE Cluster](https://cloud.google.com/kubernetes-engine)
- [Protobuf Compiler and Go Plugin](https://developers.google.com/protocol-buffers/docs/gotutorial#compiling-your-protocol-buffers)
- Go Modules listed in [go.mod](go.mod)
- [Git Submodules](../.gitmodules): `git submodule update --init`
