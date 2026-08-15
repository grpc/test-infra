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

FROM php:8.1-bullseye
RUN mkdir -p /src/workspace
WORKDIR /src/workspace

RUN apt-get update && apt-get install -y \
  git \
  zlib1g-dev \
  build-essential \
  lcov \
  make \
  gnupg2 \
  unzip \
  procps && \
  apt-get clean

# Install rvm
RUN gpg2 --recv-keys 7D2BAF1CF37B13E2069D6956105BD0E739499BDB
RUN \curl -sSL https://get.rvm.io | bash -s master

# Install Ruby 3.2
RUN /bin/bash -l -c "rvm install 3.2"
RUN /bin/bash -l -c "rvm use --default 3.2"
RUN /bin/bash -l -c "echo 'gem: --no-document' > ~/.gemrc"
RUN /bin/bash -l -c "gem install bundler --no-document -v 2.4"

# Since grpc.so dynamically links to libgrpc.so, it must be discoverable.
ENV LD_LIBRARY_PATH=/src/workspace/libs/opt

RUN mkdir /run_scripts
ADD run_worker.sh /run_scripts
ADD run_protobuf_c_worker.sh /run_scripts
RUN chmod -R 777 /run_scripts

CMD ["bash"]
