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
cd /src/workspace
ls -A | xargs -r rm -fr

# This process initializes an empty git repository, adds and fetches objects
# from the $CLONE_REPO, checks out the $CLONE_GIT_REF and then updates the
# submodules. This prevents the unnecessary checkout of the master branch.
# This process is similar to other CI systems, including GitHub actions. See:
# https://stackoverflow.com/questions/3489173.

git init
git remote add origin "${CLONE_REPO}"
git fetch origin
git checkout "${CLONE_GIT_REF}"
git submodule update --init --recursive

# At this point, the files and the directory are read-only when used with a
# Docker volume. The mode is changed to ensure that consumers of the directory
# and files over a volume have read, write and execute permissions.

chmod -R 777 .
