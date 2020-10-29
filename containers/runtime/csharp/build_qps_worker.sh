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

python tools/run_tests/run_tests.py -l csharp -c opt --build_only -j8

cd src/csharp
# even though we've already built QpsWorker by the run_tests.py comand
# above, we cannot run it as is. The problem is that only then contents
# of /src/workspace are preserved after the "build" phase finishes
# and the "run" phase starts. Since in a regular build, the nuget
# packages depended upon are downloaded into the ~/.nuget folder,
# the would be missing in the "run" phase.
# To work around this limitation, we use "dotnet publish"
# to get a self-contained QpsWorker with all its dependencies
# in a single directory.
dotnet publish Grpc.IntegrationTesting.QpsWorker/ -c Release -f netcoreapp2.1 -o ../../../qps_worker
