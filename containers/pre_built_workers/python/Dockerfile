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
RUN mkdir /executables
RUN bazel --output_user_root=/executables --bazelrc=/dev/null build --nokeep_state_after_build //src/python/grpcio_tests/tests/qps:qps_worker

RUN find /executables/ -type f ! \( -name '*.so' -o -name '*.sh'  -o -name '*.py' -o -name '*worker' \) -delete

FROM python:3.7-buster

RUN mkdir -p /pre
WORKDIR /pre
COPY --from=0 /pre/src /pre/src
COPY --from=0 /pre/bazel-bin /pre/bazel-bin
COPY --from=0 /executables /executables

RUN apt-get update && apt-get install -y \
  git \
  time && \
  apt-get clean

CMD ["bash"]
