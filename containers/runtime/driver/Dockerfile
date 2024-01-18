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

FROM marketplace.gcr.io/google/debian11

ARG REPOSITORY=grpc/grpc
ARG GITREF=master

# when BREAK_CACHE arg is set to a random value (e.g. by "--build-arg BREAK_CACHE=$(uuidgen)"),
# it makes sure the docker cache breaks at this command, and all the following
# commands in this Dockerfile will be forced to re-run on each build.
# This is important to ensure we always clone the repository even if "GITREF" stays unchanged
# (important e.g. when GITREF=master, when the clone command could get cached and
# we'd end up with a stale repository).
ARG BREAK_CACHE

RUN apt-get update && apt-get install -y git

RUN mkdir -p /src/code
WORKDIR /src/code

RUN git clone https://github.com/$REPOSITORY.git .
RUN git checkout -f $GITREF

# See https://github.com/grpc/grpc/blob/master/tools/dockerfile/README.md
FROM us-docker.pkg.dev/grpc-testing/testing-images-public/bazel:cacad91746cd598d8756de89b912be291de1f019@sha256:32bde2dcb2087f2a32afab59e4dfedf7e8c76a52c69881f63a239d311f0e5ecf

COPY --from=0 /src/code /src/code
RUN mkdir -p /tmp/build_output
WORKDIR /src/code
RUN bazel --output_user_root=/tmp/build_output build --config opt //test/cpp/qps:qps_json_driver

FROM marketplace.gcr.io/google/debian11

RUN mkdir -p /src/driver
RUN mkdir -p /src/code
RUN mkdir -p /src/workspace
WORKDIR /src/workspace

COPY --from=1 /tmp/build_output /tmp/build_output
COPY --from=1 /src/code /src/code

RUN apt-get update && apt-get upgrade -y

RUN apt-get update && apt-get install -y \
  autoconf \
  build-essential \
  clang \
  curl \
  git \
  make \
  libtool \
  libgflags-dev \
  pkg-config \
  gnupg \
  apt-transport-https \
  dnsutils \
  ca-certificates

RUN apt-get update && apt-get install -y \
  python3-dev \
  python3-pip \
  python3-setuptools \
  python3-yaml

RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | \
  tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | \
  apt-key --keyring /usr/share/keyrings/cloud.google.gpg  add - && apt-get update -y && apt-get install google-cloud-sdk -y

RUN apt-get clean

RUN pip3 install \
  protobuf \
  google-api-python-client \
  oauth2client \
  google-auth-oauthlib \
  tabulate \
  py-dateutil \
  pyasn1_modules==0.2.2 \
  pyasn1==0.4.2 \
  six==1.15.0

COPY . /src/driver
RUN chmod a+x /src/driver/run.sh

ENV QPS_WORKERS=""
ENV QPS_WORKERS_FILE=""
ENV SCENARIOS_FILE="/src/driver/example.json"
ENV BQ_RESULT_TABLE=""

CMD ["bash", "-c", "timeout --kill-after=${KILL_AFTER} ${POD_TIMEOUT} /src/driver/run.sh"]
