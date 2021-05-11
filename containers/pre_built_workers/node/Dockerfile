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

FROM node:10-buster

RUN mkdir -p /pre
WORKDIR /pre

ARG REPOSITORY=grpc/grpc-node
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


RUN mkdir /build_scripts
ADD build_qps_worker.sh /build_scripts
RUN /build_scripts/build_qps_worker.sh

RUN npm install -g pkg
ADD pkg_config.json /pre
RUN pkg -c pkg_config.json ./test/performance/worker.js

# Copy node modules to a new image
FROM debian:buster

RUN mkdir -p /pre
WORKDIR /execute

COPY --from=0 /pre/worker-linux /execute
COPY --from=0 /pre/test/fixtures/native_native.js /execute/test/fixtures/native_native.js
COPY --from=0 /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

ENV NODE_OPTIONS='--require /execute/test/fixtures/native_native.js'

CMD ["bash"]
