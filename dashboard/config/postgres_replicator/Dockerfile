# Copyright 2021 gRPC authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# syntax=docker/dockerfile:1

# Build replicator
FROM golang:1.16 AS builder

ARG REPOSITORY=grpc/test-infra
ARG GITREF=master

RUN git clone https://github.com/$REPOSITORY.git src \
    && cd src/dashboard && git checkout $GITREF \
    && make replicator REPLICATOR_OUTPUT_DIR=/

# Copy replicator binary and run it
FROM golang:1.16
WORKDIR /app
COPY --from=builder /replicator /app
COPY config.yaml /app
EXPOSE 8080

CMD [ "/app/replicator", "-c", "config.yaml" ]
