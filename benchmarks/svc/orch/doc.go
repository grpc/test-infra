// Copyright 2020 gRPC authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package orch provides a library for orchestrating sessions on
// kubernetes. Users should construct a Controller and queue sessions
// with its Schedule method. The Controller will configure many internal
// types and structures to communicate with kubernetes, monitor the
// health of test components and limit the number of running sessions.
package orch
