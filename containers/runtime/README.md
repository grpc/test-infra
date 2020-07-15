# Runtime Container Images

## [cxx](cxx/)

Base Image: [debian:stretch](https://hub.docker.com/_/debian)

Working Directory: `/src/workspace`

Entrypoint: `bash`

Official Docker Image for the Debian distribution, decorated with autoconf,
build-essential, clang, git, make, libtool, libgflags-dev and pkg-config.

## [driver](driver/)

Base Image: [debian:stretch](https://hub.docker.com/_/debian)

Working Directory: `/src/workspace`

Entrypoint: `bash`

Official Docker Image for the Debian distribution, decorated with autoconf,
build-essential, clang, git, make, libtool, libgflags-dev and pkg-config.

Includes gnupg, apt-transport-https, ca-certificates, python3-dev, python3-pip,
python3-setuptools, python3-yamlz and the Google Cloud SDK. Also installs the
protobuf, google-api-python-client, oauth2client, google-auth-oauthlib,
tabulate, six, pyasn1_modules and pyasn1 python packages.

## [go](go/)

Base Image: [golang:1.14](https://hub.docker.com/_/golang)

Entrypoint: `bash`

Working Directory: `/src/workspace`

Official Docker Image for the Go Programming Language.

Additionally installs bash, curl, git, make and time for debugging.

## [java](java/)

Base Image: [openjdk:8](https://hub.docker.com/_/openjdk)

Entrypoint: `bash`

Working Directory: `/src/workspace`

Official Docker Image of OpenJDK 8 on Debian. This is not Oracle's OpenJDK
container.

Additionally installs bash, curl, git and time for debugging.
