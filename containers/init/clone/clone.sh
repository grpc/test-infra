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

git clone --recursive $CLONE_REPO /src/workspace
if [[ $? -ne 0 ]]; then
  echo "Failed to clone repository at \"$CLONE_REPO\""
  exit $?
fi

git checkout $CLONE_GIT_REF
if [[ $? -ne 0 ]]; then
  echo "Failed to checkout git-ref \"$CLONE_GIT_REF\""
  echo "The git-ref must be a commit hash, branch or tag"
  exit $?
fi
