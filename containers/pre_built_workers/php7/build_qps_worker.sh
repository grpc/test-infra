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

export GRPC_LIB_SUBDIR="libs/opt"
export CFLAGS="-Wno-parentheses-equality"
export root="/pre"

make -j8 static_c shared_c EMBED_OPENSSL=true EMBED_ZLIB=true

cd src/php/ext/grpc

phpize
./configure --enable-grpc="/pre" --enable-coverage --enable-tests

cd /pre/src/php/tests/qps

composer install

cd ../../../../third_party/protobuf/php/ext/google/protobuf

phpize
./configure

# Prepare for ruby proxy workers
source /usr/local/rvm/scripts/rvm
mkdir -p /pre/saved/bundle/
export GEM_HOME=/pre/saved/bundle/
bundle install
rake compile
