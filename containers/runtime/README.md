# Runtime Container Images

These container images provide the environments for the load tests. They are
derivatives of the [interop test images](https://github.com/grpc/grpc/tree/master/tools/dockerfile/interoptest/).

## [cxx](cxx/)

Base Image: [Debian Stable](https://hub.docker.com/_/debian)

## [driver](driver/)

Base Image: [Debian Stable](https://hub.docker.com/_/debian)

Differs from [cxx](cxx/) by adding Google Cloud SDK and python3 interpreter.

## [go](go/)

Base Image: [Golang Image](https://hub.docker.com/_/golang)

## [java](java/)

Base Image: [Docker OpenJDK Image](https://hub.docker.com/_/openjdk)
