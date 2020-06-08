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

FROM debian:jessie

RUN apt-get update && apt-get install -y \
  autoconf \
  build-essential \
  clang \
  git \
  make \
  libtool \
  libgflags-dev \
  pkg-config

RUN apt-get update && apt-get install -y \
  python-dev \
  python-setuptools \
  python-yaml \
  python-all-dev \
  python3-all-dev \
  python-pip

RUN apt-get clean

RUN pip install --upgrade pip==19.3.1
RUN pip install virtualenv
RUN pip install futures==2.2.0 enum34==1.0.4 protobuf==3.5.2.post1 six==1.10.0 twisted==17.5.0

ARG REPOSITORY=grpc/grpc
ARG GITREF=master

RUN mkdir -p /grpc/grpc
WORKDIR /grpc/grpc

RUN git clone https://github.com/$REPOSITORY.git .
RUN git submodule update --init
COPY run.sh .
RUN chmod a+x run.sh

RUN python tools/run_tests/run_tests.py -l c++ --build_only

ENV WORKER_KIND=client
ENV DRIVER_PORT=10000
ENV SERVER_PORT=10010

EXPOSE $DRIVER_PORT
EXPOSE $SERVER_PORT

CMD ["./run.sh"]

