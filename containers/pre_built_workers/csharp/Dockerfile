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

FROM mcr.microsoft.com/dotnet/sdk:2.1

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

RUN apt-get update && apt-get install -y \
  build-essential \
  git \
  cmake && \
  apt-get clean

# Currently C# is little tricky to build with a single command, so we inject a script that can take care of the build.
RUN mkdir /build_scripts
ADD build_qps_worker.sh /build_scripts
RUN /build_scripts/build_qps_worker.sh

FROM mcr.microsoft.com/dotnet/runtime:2.1

RUN mkdir -p /execute
WORKDIR /execute
COPY --from=0 /pre/etc /execute/etc
COPY --from=0 /pre/qps_worker /execute/qps_worker
COPY --from=0 /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

CMD ["bash"]
