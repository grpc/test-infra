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


if [ -f /var/data/qps_workers/server_target_override ]; then
  SERVER_TARGET_OVERRIDE=$(cat /var/data/qps_workers/server_target_override)
fi

/src/code/bazel-bin/test/cpp/qps/qps_json_driver --scenarios_file="${SCENARIOS_FILE}" \
  --scenario_result_file=scenario_result.json --qps_server_target_override="${SERVER_TARGET_OVERRIDE}"

/src/code/bazel-bin/test/cpp/qps/qps_json_driver --quit=true

if [ -n "${BQ_RESULT_TABLE}" ]; then
  if [ -r "${METADATA_OUTPUT_FILE}" ]; then
    cp "${METADATA_OUTPUT_FILE}" metadata.json
  fi
  if [ -r "${NODE_INFO_OUTPUT_FILE}" ]; then
    cp "${NODE_INFO_OUTPUT_FILE}" node_info.json
    if [ -n "${SERVER_TARGET_OVERRIDE}" ] || [ -n "${ENABLE_PROMETHEUS}" ]; then
      if  [ "$(dig +short -t srv prometheus.test-infra-system.svc.cluster.local)" ]; then
        python3 /src/code/tools/run_tests/performance/prometheus.py \
          --url=http://prometheus.test-infra-system.svc.cluster.local:9090 \
          --pod_type=clients --container_name=main \
          --container_name=sidecar --delay_seconds=20
      fi
    fi
  fi
  python3 /src/code/tools/run_tests/performance/bq_upload_result.py --bq_result_table="${BQ_RESULT_TABLE}"
fi
