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
  curl \
  git \
  make \
  time

RUN echo "deb http://archive.debian.org/debian/ jessie-backports main contrib non-free" > /etc/apt/sources.list.d/jessie-backports.list && \
    echo 'Acquire::Check-Valid-Until no;' > /etc/apt/apt.conf.d/99no-check-valid-until && \
    apt-get update && \
    apt-get install -y --no-install-recommends -t jessie-backports openjdk-8-jdk-headless

RUN apt-get clean

RUN mkdir -p /grpc/grpc-java
WORKDIR /grpc/grpc-java

ARG REPOSITORY=grpc/grpc-java
ARG GITREF=master

RUN git clone https://github.com/$REPOSITORY.git .
RUN git checkout $GITREF
RUN ./gradlew --no-daemon -PskipCodegen=true -PskipAndroid=true :grpc-benchmarks:installDist
COPY run.sh .
RUN chmod a+x run.sh

ENV WORKER_KIND=client
ENV DRIVER_PORT=10000
ENV SERVER_PORT=10010

EXPOSE $DRIVER_PORT
EXPOSE $SERVER_PORT

CMD ["./run.sh"]

