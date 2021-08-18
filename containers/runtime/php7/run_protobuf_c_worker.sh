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

while getopts s: flag; do
  case "${flag}" in
  s) input_server_port=${OPTARG} ;;
  *) echo "usage: $0 -s [server_port]" >&1
       exit 1 ;;
  esac
done

echo "Server port: ${input_server_port}"

source /usr/local/rvm/scripts/rvm
export GEM_HOME=/src/workspace/saved/bundle/

if [ -z "${input_server_port}" ]; then
  echo "Server port is not set, starting the worker without server port provided"
  ruby src/ruby/qps/proxy-worker.rb \
    --use_protobuf_c_extension \
    --driver_port="${DRIVER_PORT}"
else
  echo "Server port: ${input_server_port}"
  ruby src/ruby/qps/proxy-worker.rb \
    --use_protobuf_c_extension \
    --driver_port="${DRIVER_PORT}" \
    --server_port="${input_server_port}"
fi
