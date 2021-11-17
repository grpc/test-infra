# Copyright 2020 gRPC authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM mcr.microsoft.com/dotnet/sdk:6.0-bullseye-slim

# TODO: when running on kokoro, this build step will not be cached
# since we'll always be on a fresh VM. Re-running this command each
# time leads to increased latency and flakiness.
RUN apt-get update && apt-get install -y \
  build-essential \
  git \
  cmake && \
  apt-get clean

RUN mkdir -p /pre
WORKDIR /pre

ARG REPOSITORY=grpc/grpc
ARG GITREF=master
# when BREAK_CACHE arg is set to a random value (e.g. by "--build-arg BREAK_CACHE=$(uuidgen)"),
# it makes sure the docker cache breaks at this command, and all the following
# commands in this Dockerfile will be forced to re-run on each build.
# This is important to ensure we always clone the repository even if "GITREF" stays unchanged
# (important e.g. when GITREF=master, when the clone command could get cached and
# we'd end up with a stale repository).
ARG BREAK_CACHE

RUN git clone https://github.com/$REPOSITORY.git .
RUN git checkout $GITREF
RUN git submodule update --init

# Save commit sha for debug use
RUN echo 'COMMIT SHA' > GRPC_GIT_COMMIT.txt
RUN git rev-parse $GITREF >> GRPC_GIT_COMMIT.txt

# Currently C# is little tricky to build with a single command, so we inject a script that can take care of the build.
RUN mkdir /build_scripts
ADD build_qps_worker.sh /build_scripts
RUN /build_scripts/build_qps_worker.sh

# Note that the libgrpc_csharp_ext.so library built by the build image
# needs to be binary-compatible with the runtime image (e.g. they need to be based off of the same linux distro)
# Because of that, care needs to be taken to always used a runtime image that's compatible with the build image.
FROM mcr.microsoft.com/dotnet/runtime:3.1-bullseye-slim

RUN mkdir -p /execute
WORKDIR /execute
COPY --from=0 /pre/etc /execute/etc
COPY --from=0 /pre/qps_worker /execute/qps_worker
COPY --from=0 /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

CMD ["bash"]
