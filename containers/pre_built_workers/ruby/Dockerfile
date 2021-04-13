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

FROM ruby:2.5

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

ENV GEM_HOME=/pre/vendor/bundle/
RUN apt-get update && apt-get install -y cmake && apt-get clean

RUN mkdir /build_scripts
ADD build_qps_worker.sh /build_scripts
RUN bash /build_scripts/build_qps_worker.sh

# Copy node modules to a new image
FROM ruby:2.5

RUN mkdir -p /execute
WORKDIR /execute
COPY --from=0 /pre/src /execute/src
COPY --from=0 /pre/vendor/bundle /execute/vendor/bundle
COPY --from=0 /pre/etc /execute/etc
COPY --from=0 /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

ENV GEM_HOME=/execute/vendor/bundle/

CMD ["bash"]
