// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package querier provides tools to interact with the Minder database
package querier

import (
	"context"

	"github.com/google/uuid"

	dsservice "github.com/mindersec/minder/internal/datasources/service"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// DataSourceHandlers interface provides functions to interact with data sources
type DataSourceHandlers interface {
	CreateDataSource(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		dataSource *pb.DataSource,
	) (*pb.DataSource, error)
	GetDataSourceByName(
		ctx context.Context,
		projectID uuid.UUID,
		name string,
	) (*pb.DataSource, error)
	UpdateDataSource(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		dataSource *pb.DataSource,
	) (*pb.DataSource, error)
	DeleteDataSource(
		ctx context.Context,
		projectID uuid.UUID,
		dataSourceID uuid.UUID,
	) error
}

// CreateDataSource creates a data source
func (q *querierType) CreateDataSource(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	dataSource *pb.DataSource,
) (*pb.DataSource, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	if q.dataSourceSvc == nil {
		return nil, ErrDataSourceSvcMissing
	}
	return q.dataSourceSvc.Create(ctx, projectID, subscriptionID, dataSource, dsservice.OptionsBuilder().WithTransaction(q.querier))
}

// UpdateDataSource updates a data source
func (q *querierType) UpdateDataSource(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	dataSource *pb.DataSource,
) (*pb.DataSource, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	if q.dataSourceSvc == nil {
		return nil, ErrDataSourceSvcMissing
	}
	return q.dataSourceSvc.Update(ctx, projectID, subscriptionID, dataSource, dsservice.OptionsBuilder().WithTransaction(q.querier))
}

// GetDataSourceByName returns a data source by name and project IDs
func (q *querierType) GetDataSourceByName(ctx context.Context, projectID uuid.UUID, name string) (*pb.DataSource, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	if q.dataSourceSvc == nil {
		return nil, ErrDataSourceSvcMissing
	}
	return q.dataSourceSvc.GetByName(ctx, name, projectID, dsservice.ReadBuilder().WithTransaction(q.querier))
}

// DeleteDataSource deletes a data source
func (q *querierType) DeleteDataSource(ctx context.Context, projectID uuid.UUID, dataSourceID uuid.UUID) error {
	if q.querier == nil {
		return ErrQuerierMissing
	}
	if q.dataSourceSvc == nil {
		return ErrDataSourceSvcMissing
	}
	return q.dataSourceSvc.Delete(ctx, projectID, dataSourceID, dsservice.OptionsBuilder().WithTransaction(q.querier))
}
