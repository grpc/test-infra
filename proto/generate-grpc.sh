#!/bin/bash
# Copyright 2022 gRPC authors
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

set -e

declare -a protos=(
    grpc/core/stats.proto
    grpc/testing/messages.proto
    grpc/testing/control.proto
    grpc/testing/report_qps_scenario_service.proto
    grpc/testing/worker_service.proto
    grpc/testing/stats.proto
    grpc/testing/benchmark_service.proto
    grpc/testing/payloads.proto
)

declare -A package=(
    [grpc/core]=grpc_core
    [grpc/testing]=grpc_testing
)

for proto in "${protos[@]}"; do
    opts[${#opts[@]}]="--go_opt=M${proto}=github.com/grpc/test-infra/proto/${package[${proto%/*}]}"
done

protoc -I../third_party/grpc-proto --go_out=. \
        --go_opt=module=github.com/grpc/test-infra/proto \
        "${opts[@]}" "${protos[@]}"
