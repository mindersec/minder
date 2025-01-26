// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/datasources/service"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/flags"
	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// CreateDataSource creates a data source
func (s *Server) CreateDataSource(ctx context.Context,
	in *minderv1.CreateDataSourceRequest) (*minderv1.CreateDataSourceResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	entityCtx := engcontext.EntityFromContext(ctx)
	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "error in entity context: %v", err)
	}

	projectID := entityCtx.Project.ID

	// Get the data source from the request
	dsReq := in.GetDataSource()
	if dsReq == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing data source")
	}

	if err := s.forceDataSourceProject(ctx, dsReq); err != nil {
		return nil, err
	}

	// Process the request
	ret, err := s.dataSourcesService.Create(ctx, projectID, uuid.Nil, dsReq, nil)
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.CreateDataSourceResponse{DataSource: ret}, nil
}

// GetDataSourceById retrieves a data source by ID
func (s *Server) GetDataSourceById(ctx context.Context,
	in *minderv1.GetDataSourceByIdRequest) (*minderv1.GetDataSourceByIdResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	// Get the data source ID from the request
	dsIDstr := in.GetId()
	if dsIDstr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing data source ID")
	}

	// Parse the data source ID
	dsID, err := uuid.Parse(dsIDstr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid data source ID: %v", err)
	}

	// Get the project ID from the request context
	entityCtx := engcontext.EntityFromContext(ctx)

	// Ensure the project is valid and exist in the db
	err = entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// Get the data source by ID
	ds, err := s.dataSourcesService.GetByID(ctx, dsID, entityCtx.Project.ID, &service.ReadOptions{})
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.GetDataSourceByIdResponse{DataSource: ds}, nil
}

// GetDataSourceByName retrieves a data source by name
func (s *Server) GetDataSourceByName(ctx context.Context,
	in *minderv1.GetDataSourceByNameRequest) (*minderv1.GetDataSourceByNameResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	// Get the data source name from the request
	dsName := in.GetName()
	if dsName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing data source name")
	}

	// Get the project ID from the request context
	entityCtx := engcontext.EntityFromContext(ctx)

	// Ensure the project is valid and exist in the db
	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// Get the data source by name
	ds, err := s.dataSourcesService.GetByName(ctx, dsName, entityCtx.Project.ID, &service.ReadOptions{})
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.GetDataSourceByNameResponse{DataSource: ds}, nil
}

// ListDataSources lists all data sources
func (s *Server) ListDataSources(ctx context.Context,
	_ *minderv1.ListDataSourcesRequest) (*minderv1.ListDataSourcesResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	// Get the project ID from the request context
	entityCtx := engcontext.EntityFromContext(ctx)

	// Ensure the project is valid and exist in the db
	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// Get all data sources
	ret, err := s.dataSourcesService.List(ctx, entityCtx.Project.ID, &service.ReadOptions{})
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.ListDataSourcesResponse{DataSources: ret}, nil
}

// UpdateDataSource updates a data source
func (s *Server) UpdateDataSource(ctx context.Context,
	in *minderv1.UpdateDataSourceRequest) (*minderv1.UpdateDataSourceResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	entityCtx := engcontext.EntityFromContext(ctx)
	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	projectID := entityCtx.Project.ID

	// Get the data source from the request
	dsReq := in.GetDataSource()
	if dsReq == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing data source")
	}

	if err := s.forceDataSourceProject(ctx, dsReq); err != nil {
		return nil, err
	}

	// Process the request
	ret, err := s.dataSourcesService.Update(ctx, projectID, uuid.Nil, dsReq, nil)
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.UpdateDataSourceResponse{DataSource: ret}, nil
}

// DeleteDataSourceById deletes a data source by ID
func (s *Server) DeleteDataSourceById(ctx context.Context,
	in *minderv1.DeleteDataSourceByIdRequest) (*minderv1.DeleteDataSourceByIdResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	// Get the data source ID from the request
	dsIDstr := in.GetId()
	if dsIDstr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing data source ID")
	}

	// Parse the data source ID
	dsID, err := uuid.Parse(dsIDstr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid data source ID: %v", err)
	}

	// Get the project ID from the request context
	entityCtx := engcontext.EntityFromContext(ctx)

	// Ensure the project is valid and exist in the db
	err = entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// Delete the data source by ID
	err = s.dataSourcesService.Delete(ctx, dsID, entityCtx.Project.ID, nil)
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.DeleteDataSourceByIdResponse{Id: dsIDstr}, nil
}

// DeleteDataSourceByName deletes a data source by name
func (s *Server) DeleteDataSourceByName(ctx context.Context,
	in *minderv1.DeleteDataSourceByNameRequest) (*minderv1.DeleteDataSourceByNameResponse, error) {

	// Check if the DataSources feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.DataSources) {
		return nil, status.Errorf(codes.Unavailable, "DataSources feature is disabled")
	}

	// Get the data source name from the request
	dsName := in.GetName()
	if dsName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing data source name")
	}

	// Get the project ID from the request context
	entityCtx := engcontext.EntityFromContext(ctx)

	// Ensure the project is valid and exist in the db
	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// Get the data source id by name
	ds, err := s.dataSourcesService.GetByName(ctx, dsName, entityCtx.Project.ID, &service.ReadOptions{})
	if err != nil {
		return nil, err
	}

	// Parse the data source ID
	dsId, err := uuid.Parse(ds.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid data source ID: %v", err)
	}

	// Delete the data source by its ID after getting it by name
	err = s.dataSourcesService.Delete(ctx, dsId, entityCtx.Project.ID, nil)
	if err != nil {
		return nil, err
	}

	// Return the response
	return &minderv1.DeleteDataSourceByNameResponse{Name: dsName}, nil
}

func (s *Server) forceDataSourceProject(ctx context.Context, in *minderv1.DataSource) error {
	entityCtx := engcontext.EntityFromContext(ctx)

	// Ensure the project is valid and exist in the db
	if err := entityCtx.ValidateProject(ctx, s.store); err != nil {
		return status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// Force the context to have the observed project ID
	if in.GetContext() == nil {
		in.Context = &minderv1.ContextV2{}
	}
	in.GetContext().ProjectId = entityCtx.Project.ID.String()

	return nil
}
