//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

// Package controlplane contains the gRPC server implementation for the control plane
package controlplane

import (
	"context"

	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// PaginationLimit is the maximum number of items that can be returned in a single page
const PaginationLimit = 10

// CheckHealth is a simple health check for monitoring
// The lintcheck is disabled because the unused-receiver is required by
// the implementation. UnimplementedHealthServiceServer is initialized
// within the Server struct
//
//revive:disable:unused-receiver
func (s *Server) CheckHealth(_ context.Context, _ *pb.CheckHealthRequest) (*pb.CheckHealthResponse, error) {
	return &pb.CheckHealthResponse{Status: "OK"}, nil
}
