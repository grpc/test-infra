#!/bin/bash
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

set -ex

if [ -n "${QPS_WORKERS_FILE}" ]; then
  export QPS_WORKERS=$(cat "${QPS_WORKERS_FILE}")
fi

/src/code/bazel-bin/test/cpp/qps/qps_json_driver --scenarios_file="${SCENARIOS_FILE}" \
  --scenario_result_file=scenario_result.json

/src/code/bazel-bin/test/cpp/qps/qps_json_driver --quit=true

if [ -n "${BQ_RESULT_TABLE}" ]; then
  if [ -r "${METADATA_OUTPUT_FILE}" ]; then
    cp "${METADATA_OUTPUT_FILE}" metadata.json
  fi
  if [ -r "${NODE_INFO_OUTPUT_FILE}" ]; then
    cp "${NODE_INFO_OUTPUT_FILE}" node_info.json
  fi
  python3 /src/code/tools/run_tests/performance/bq_upload_result.py --bq_result_table="${BQ_RESULT_TABLE}"
fi
