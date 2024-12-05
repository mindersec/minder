// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package controlplane contains the gRPC server implementation for the control plane
package controlplane

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
