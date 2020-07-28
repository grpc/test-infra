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
// as, their generated code. In addition, it contains generated code for
// the gRPC third party protos.
//
// The protos can be rebuilt using the `go generate` command.
package proto

// Build grpc/core package
//go:generate protoc -I ../third_party/grpc-proto --go_out=paths=source_relative,plugins=grpc:. grpc/core/stats.proto

// Build grpc/testing package
//go:generate protoc -I ../third_party/grpc-proto --go_out=paths=source_relative,Mgrpc/core/stats.proto=github.com/grpc/test-infra/proto/grpc/core,plugins=grpc:. grpc/testing/benchmark_service.proto grpc/testing/control.proto grpc/testing/messages.proto grpc/testing/payloads.proto grpc/testing/report_qps_scenario_service.proto grpc/testing/stats.proto grpc/testing/worker_service.proto
