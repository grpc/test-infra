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

FROM openjdk:11-jdk-bullseye

RUN mkdir -p /pre
WORKDIR /pre

ARG REPOSITORY=grpc/grpc-java
ARG GITREF=master
# when BREAK_CACHE arg is set to a random value (e.g. by "--build-arg BREAK_CACHE=$(uuidgen)"),
# it makes sure the docker cache breaks at this command, and all the following
# commands in this Dockerfile will be forced to re-run on each build.
# This is important to ensure we always clone the repository even if "GITREF" stays unchanged
# (important e.g. when GITREF=master, when the clone command could get cached and
# we'd end up with a stale repository).
ARG BREAK_CACHE

RUN git clone https://github.com/$REPOSITORY.git .
RUN git checkout $GITREF
RUN git submodule update --init

# Save commit sha for debug use
RUN echo 'COMMIT SHA' > GRPC_GIT_COMMIT.txt
RUN git rev-parse $GITREF >> GRPC_GIT_COMMIT.txt

RUN ./gradlew -PskipAndroid=true -PskipCodegen=true -Dorg.gradle.jvmargs=-Xmx2g :grpc-benchmarks:installDist

FROM openjdk:11-jdk-slim-bullseye

RUN mkdir -p /execute
WORKDIR /execute
COPY --from=0 /pre/benchmarks/build/install/grpc-benchmarks /execute
COPY --from=0 /pre/GRPC_GIT_COMMIT.txt /execute/GRPC_GIT_COMMIT.txt

RUN mkdir /run_scripts
ADD run_worker.sh /run_scripts
RUN chmod -R 777 /run_scripts

CMD ["bash"]
