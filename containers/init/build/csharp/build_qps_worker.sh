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

# Build C# native extension with cmake
mkdir -p cmake/build
pushd cmake/build
cmake -DgRPC_BUILD_TESTS=OFF -DCMAKE_BUILD_TYPE=Release ../..
make -j8 grpc_csharp_ext
popd

cd src/csharp
dotnet publish Grpc.IntegrationTesting.QpsWorker/ -c Release -f netcoreapp3.1 -o ../../qps_worker
