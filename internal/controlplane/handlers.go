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

// Package controlplane contains the gRPC server implementation for the control plane
package controlplane

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// PaginationLimit is the maximum number of items that can be returned in a single page
const PaginationLimit = 10

// CheckHealth is a simple health check for monitoring
func (s *Server) CheckHealth(ctx context.Context, _ *pb.CheckHealthRequest) (*pb.CheckHealthResponse, error) {
	if err := s.store.CheckHealth(); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("health check failed")
		return nil, status.Errorf(codes.Internal, "failed to check health: %v", err)
	}
	return &pb.CheckHealthResponse{Status: "OK"}, nil
}
