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

// Package proto contains the proto definitions for the service, as well
// as, its generated code.
//
// The protos can be rebuilt using the `go generate` command.
package proto

// Build benchmarks service proto
//go:generate protoc -I ../../third_party/grpc-proto -I ../../third_party/googleapis -I . --go_out=paths=source_relative,plugins=grpc,Mgrpc/testing/control.proto=github.com/grpc/test-infra/proto/grpc/testing,Mgoogle/longrunning/operations.proto=google.golang.org/genproto/googleapis/longrunning:. scheduling/v1/scheduling_service.proto
