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

// Generate endpointupdater package
//go:generate protoc -Iendpointupdater --go_out=endpointupdater --go-grpc_out=endpointupdater --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative endpointupdater/endpoint.proto

// Generate grpc_core and grpc_testing packages
//go:generate ./generate-grpc.sh
