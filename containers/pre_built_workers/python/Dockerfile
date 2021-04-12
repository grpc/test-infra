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

FROM l.gcr.io/google/bazel:latest

RUN mkdir -p /pre
WORKDIR /pre

ARG REPOSITORY=grpc/grpc
ARG GITREF=master

RUN git clone https://github.com/$REPOSITORY.git .
RUN git submodule update --init
RUN git checkout $GITREF

# Save commit sha for debug use
RUN echo 'COMMIT SHA' > GRPC_GIT_COMMIT.txt
RUN git rev-parse $GITREF >> GRPC_GIT_COMMIT.txt

RUN mkdir /executables
RUN bazel --output_user_root=/executables build --nobuild_runfile_links --nobuild_runfile_manifests --build_python_zip //src/python/grpcio_tests/tests/qps:qps_worker

FROM python:3.7-buster

RUN mkdir -p /execute
WORKDIR /execute
COPY --from=0 /pre/bazel-bin/src/python/grpcio_tests/tests/qps/qps_worker /execute/qps_worker
COPY --from=0 /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

RUN apt-get update && apt-get install -y \
  git \
  time && \
  apt-get clean

CMD ["bash"]
