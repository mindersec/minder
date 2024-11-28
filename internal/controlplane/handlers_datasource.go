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
	in *minderv1.CreateDataSourceRequest) (*minderv1.CreateDataSourceResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	// Get the data source from the request
	dsReq := in.GetDataSource()
	if dsReq == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing data source")
	}

	// Process the request
	ret, err := s.dataSourcesService.Create(ctx, dsReq, nil)
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.CreateDataSourceResponse{DataSource: ret}, nil
}

// GetDataSourceById retrieves a data source by ID
func (s *Server) GetDataSourceById(ctx context.Context,
	_ *minderv1.GetDataSourceByIdRequest) (*minderv1.GetDataSourceByIdResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.GetDataSourceByIdResponse{}, nil
}

// GetDataSourceByName retrieves a data source by name
func (s *Server) GetDataSourceByName(ctx context.Context,
	_ *minderv1.GetDataSourceByNameRequest) (*minderv1.GetDataSourceByNameResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.GetDataSourceByNameResponse{}, nil
}

// ListDataSources lists all data sources
func (s *Server) ListDataSources(ctx context.Context,
	_ *minderv1.ListDataSourcesRequest) (*minderv1.ListDataSourcesResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.ListDataSourcesResponse{}, nil
}

// UpdateDataSource updates a data source
func (s *Server) UpdateDataSource(ctx context.Context,
	_ *minderv1.UpdateDataSourceRequest) (*minderv1.UpdateDataSourceResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.UpdateDataSourceResponse{}, nil
}

// DeleteDataSourceById deletes a data source by ID
func (s *Server) DeleteDataSourceById(ctx context.Context,
	_ *minderv1.DeleteDataSourceByIdRequest) (*minderv1.DeleteDataSourceByIdResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.DeleteDataSourceByIdResponse{}, nil
}

// DeleteDataSourceByName deletes a data source by name
func (s *Server) DeleteDataSourceByName(ctx context.Context,
	_ *minderv1.DeleteDataSourceByNameRequest) (*minderv1.DeleteDataSourceByNameResponse, error) {

	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	return &minderv1.DeleteDataSourceByNameResponse{}, nil
}
