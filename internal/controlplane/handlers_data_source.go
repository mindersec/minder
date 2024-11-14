// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/flags"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// CreateDataSource creates a data source
func (s *Server) CreateDataSource(ctx context.Context,
	req *minderv1.CreateDataSourceRequest) (*minderv1.CreateDataSourceResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.CreateDataSourceResponse{}, nil
}

// GetDataSourceById retrieves a data source by ID
func (s *Server) GetDataSourceById(ctx context.Context,
	req *minderv1.GetDataSourceByIdRequest) (*minderv1.GetDataSourceByIdResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.GetDataSourceByIdResponse{}, nil
}

// ListDataSources lists all data sources
func (s *Server) ListDataSources(ctx context.Context,
	req *minderv1.ListDataSourcesRequest) (*minderv1.ListDataSourcesResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.ListDataSourcesResponse{}, nil
}

// UpdateDataSource updates a data source
func (s *Server) UpdateDataSource(ctx context.Context,
	req *minderv1.UpdateDataSourceRequest) (*minderv1.UpdateDataSourceResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.UpdateDataSourceResponse{}, nil
}

// DeleteDataSource deletes a data source
func (s *Server) DeleteDataSource(ctx context.Context,
	req *minderv1.DeleteDataSourceRequest) (*minderv1.DeleteDataSourceResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.DeleteDataSourceResponse{}, nil
}
