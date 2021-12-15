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

FROM golang:1.16-alpine
WORKDIR /app

# Copy config
COPY config.yaml ./
COPY test-infra ./test-infra

ENV CGO_ENABLED=0
RUN apk add make
RUN cd test-infra \
    && go mod download \
    && cd dashboard && make replicator REPLICATOR_OUTPUT_DIR=/
RUN rm -rf test-infra

EXPOSE 8080
CMD [ "/replicator", "-c", "config.yaml" ]
