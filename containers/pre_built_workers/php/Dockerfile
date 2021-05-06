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

FROM php:7.2.22-buster

RUN mkdir -p /pre
WORKDIR /pre

RUN apt-get update && apt-get install -y \
  git \
  zlib1g-dev \
  build-essential \
  lcov \
  make \
  gnupg2 && \
  apt-get clean

ARG REPOSITORY=grpc/grpc
ARG GITREF=master

RUN git clone https://github.com/$REPOSITORY.git .
RUN git checkout $GITREF
RUN git submodule update --init

# Save commit sha for debug use
RUN echo 'COMMIT SHA' > GRPC_GIT_COMMIT.txt
RUN git rev-parse $GITREF >> GRPC_GIT_COMMIT.txt

# Install rvm
RUN gpg2 --keyserver hkp://pool.sks-keyservers.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB
RUN \curl -sSL https://get.rvm.io | bash -s stable

# Install Ruby 2.5
RUN apt-get update && apt-get install -y procps && apt-get clean
RUN /bin/bash -l -c "rvm install ruby-2.5"
RUN /bin/bash -l -c "rvm use --default ruby-2.5"
RUN /bin/bash -l -c "echo 'gem: --no-document' > ~/.gemrc"
RUN /bin/bash -l -c "echo 'export PATH=/usr/local/rvm/bin:$PATH' >> ~/.bashrc"
RUN /bin/bash -l -c "echo 'rvm --default use ruby-2.5' >> ~/.bashrc"
RUN /bin/bash -l -c "gem install bundler --no-document -v 1.9"

# Install composer
RUN curl -sS https://getcomposer.org/installer | php
RUN mv composer.phar /usr/local/bin/composer

RUN mkdir /build_scripts
ADD build_qps_worker.sh /build_scripts
RUN /build_scripts/build_qps_worker.sh

FROM php:7.2.22-buster

RUN mkdir -p /execute
WORKDIR /execute
COPY --from=0 /pre/src /execute/src
COPY --from=0 /pre/etc /execute/etc
COPY --from=0 /pre/saved/bundle/ /execute/saved/bundle/
COPY --from=0 /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

RUN apt-get update && apt-get install -y \
  zlib1g-dev \
  build-essential \
  lcov \
  make \
  gnupg2 \
  procps && \
  apt-get clean

# Install rvm
RUN gpg2 --keyserver hkp://pool.sks-keyservers.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB
RUN \curl -sSL https://get.rvm.io | bash -s stable

# Install Ruby 2.5
RUN /bin/bash -l -c "rvm install ruby-2.5"
RUN /bin/bash -l -c "rvm use --default ruby-2.5"
RUN /bin/bash -l -c "echo 'gem: --no-document' > ~/.gemrc"
RUN /bin/bash -l -c "echo 'export PATH=/usr/local/rvm/bin:$PATH' >> ~/.bashrc"
RUN /bin/bash -l -c "echo 'rvm --default use ruby-2.5' >> ~/.bashrc"
RUN /bin/bash -l -c "gem install bundler --no-document -v 1.9"

RUN mkdir /run_scripts
ADD run_worker.sh /run_scripts
CMD ["bash"]
