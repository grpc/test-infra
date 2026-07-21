# Copyright 2026 gRPC authors
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

FROM rust:1-bookworm AS builder

RUN apt-get update && apt-get install -y cmake && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /executable
WORKDIR /executable

# TODO: arjan-bal - point to official repo once https://github.com/grpc/grpc-rust/pull/2737
# is merged.
ARG REPOSITORY=arjan-bal/tonic
ARG GITREF=grpc-benchmark-client
# when BREAK_CACHE arg is set to a random value (e.g. by "--build-arg BREAK_CACHE=$(uuidgen)"),
# it makes sure the docker cache breaks at this command, and all the following
# commands in this Dockerfile will be forced to re-run on each build.
# This is important to ensure we always clone the repository even if "GITREF" stays unchanged
# (important e.g. when GITREF=master, when the clone command could get cached and
# we'd end up with a stale repository).
ARG BREAK_CACHE

RUN git clone https://github.com/$REPOSITORY.git .
RUN git checkout $GITREF

# Save commit sha for debug use
RUN echo 'COMMIT SHA' > GRPC_GIT_COMMIT.txt
RUN git rev-parse $GITREF >> GRPC_GIT_COMMIT.txt

RUN  cargo build -p grpc-benchmark --release --bin worker

# Copy executable to a new image
FROM debian:bookworm

RUN mkdir -p /executable
WORKDIR /executable
COPY --from=0 /executable/GRPC_GIT_COMMIT.txt /executable/GRPC_GIT_COMMIT.txt
COPY --from=0 /executable/target/release/worker /executable/bin/worker

CMD ["bash"]
