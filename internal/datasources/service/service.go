// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service encodes the business logic for dealing with data sources.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/mindersec/minder/internal/datasources"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// DataSourcesService is an interface that defines the methods for the data sources service.
type DataSourcesService interface {
	// GetByName returns a data source by name.
	GetByName(ctx context.Context, name string, project uuid.UUID, opts *ReadOptions) (*minderv1.DataSource, error)

	// GetByID returns a data source by ID.
	GetByID(ctx context.Context, id uuid.UUID, project uuid.UUID, opts *ReadOptions) (*minderv1.DataSource, error)

	// List lists all data sources in the given project.
	List(ctx context.Context, project uuid.UUID, opts *ReadOptions) ([]*minderv1.DataSource, error)

	// Create creates a new data source.
	Create(ctx context.Context, ds *minderv1.DataSource, opts *Options) (*minderv1.DataSource, error)

	// Update updates an existing data source.
	Update(ctx context.Context, ds *minderv1.DataSource, opts *Options) (*minderv1.DataSource, error)

	// Delete deletes a data source in the given project.
	//
	// Note that one cannot delete a data source that is in use by a rule type.
	Delete(ctx context.Context, id uuid.UUID, project uuid.UUID, opts *Options) error

	// ValidateRuleTypeReferences takes the data source declarations in
	// a rule type and validates that the data sources are available
	// in the project hierarchy.
	//
	// Note that the rule type already contains project information.
	ValidateRuleTypeReferences(ctx context.Context, rt *minderv1.RuleType, opts *Options) error

	// BuildDataSourceRegistry bundles up all data sources referenced in the rule type
	// into a registry.
	BuildDataSourceRegistry(ctx context.Context, rt *minderv1.RuleType, opts *Options) (*v1datasources.DataSourceRegistry, error)
}

type dataSourceService struct {
	store db.Store

	// This is a function that will begin a transaction for the service.
	// We make this a function so that we can mock it in tests.
	txBuilder func(d *dataSourceService, opts txGetter) (serviceTX, error)
}

// NewDataSourceService creates a new data source service.
func NewDataSourceService(store db.Store) *dataSourceService {
	return &dataSourceService{
		store:     store,
		txBuilder: beginTx,
	}
}

// WithTransactionBuilder sets the transaction builder for the data source service.
//
// Note this is mostly just useful for testing.
func (d *dataSourceService) WithTransactionBuilder(txBuilder func(d *dataSourceService, opts txGetter) (serviceTX, error)) {
	d.txBuilder = txBuilder
}

// Ensure that dataSourceService implements DataSourcesService.
var _ DataSourcesService = (*dataSourceService)(nil)

func (d *dataSourceService) GetByName(
	ctx context.Context, name string, project uuid.UUID, opts *ReadOptions) (*minderv1.DataSource, error) {
	return d.getDataSourceSomehow(
		ctx, project, opts, func(ctx context.Context, tx db.ExtendQuerier, projs []uuid.UUID,
		) (db.DataSource, error) {
			ds, err := tx.GetDataSourceByName(ctx, db.GetDataSourceByNameParams{
				Name:     name,
				Projects: projs,
			})
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return db.DataSource{}, util.UserVisibleError(codes.NotFound,
						"data source of name %s not found", name)
				}
				return db.DataSource{}, fmt.Errorf("failed to get data source by name: %w", err)
			}

			return ds, nil
		})
}

func (d *dataSourceService) GetByID(
	ctx context.Context, id uuid.UUID, project uuid.UUID, opts *ReadOptions) (*minderv1.DataSource, error) {
	return d.getDataSourceSomehow(
		ctx, project, opts, func(ctx context.Context, tx db.ExtendQuerier, projs []uuid.UUID,
		) (db.DataSource, error) {
			ds, err := tx.GetDataSource(ctx, db.GetDataSourceParams{
				ID:       id,
				Projects: projs,
			})
			if errors.Is(err, sql.ErrNoRows) {
				return db.DataSource{}, util.UserVisibleError(codes.NotFound,
					"data source of id %s not found", id.String())
			}
			if err != nil {
				return db.DataSource{}, fmt.Errorf("failed to get data source by name: %w", err)
			}

			return ds, nil
		})
}

func (d *dataSourceService) List(
	ctx context.Context, project uuid.UUID, opts *ReadOptions) ([]*minderv1.DataSource, error) {
	stx, err := d.txBuilder(d, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	//nolint:gosec // we'll log this error later.
	defer stx.Rollback()

	tx := stx.Q()

	projs, err := listRelevantProjects(ctx, tx, project, opts.canSearchHierarchical())
	if err != nil {
		return nil, fmt.Errorf("failed to list relevant projects: %w", err)
	}

	dss, err := tx.ListDataSources(ctx, projs)
	if err != nil {
		return nil, fmt.Errorf("failed to list data sources: %w", err)
	}

	outDS := make([]*minderv1.DataSource, len(dss))

	for i, ds := range dss {
		dsfuncs, err := tx.ListDataSourceFunctions(ctx, db.ListDataSourceFunctionsParams{
			DataSourceID: ds.ID,
			ProjectID:    ds.ProjectID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list data source functions: %w", err)
		}

		dsProtobuf, err := dataSourceDBToProtobuf(ds, dsfuncs)
		if err != nil {
			return nil, fmt.Errorf("failed to convert data source to protobuf: %w", err)
		}

		outDS[i] = dsProtobuf
	}

	if err := stx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return outDS, nil
}

func (d *dataSourceService) Create(
	ctx context.Context, ds *minderv1.DataSource, opts *Options) (*minderv1.DataSource, error) {
	if ds == nil {
		return nil, errors.New("data source is nil")
	}

	stx, err := d.txBuilder(d, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func(stx serviceTX) {
		err := stx.Rollback()
		if err != nil {
			fmt.Printf("failed to rollback transaction: %v", err)
		}
	}(stx)

	tx := stx.Q()

	projectID, err := uuid.Parse(ds.GetContext().GetProjectId())
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Check if such data source  already exists in project hierarchy
	projs, err := listRelevantProjects(ctx, tx, projectID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list relevant projects: %w", err)
	}
	existing, err := tx.GetDataSourceByName(ctx, db.GetDataSourceByNameParams{
		Name:     ds.GetName(),
		Projects: projs,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check for existing data source: %w", err)
	}
	if existing.ID != uuid.Nil {
		return nil, util.UserVisibleError(codes.AlreadyExists,
			"data source with name %s already exists", ds.GetName())
	}

	// Create data source record
	dsRecord, err := tx.CreateDataSource(ctx, db.CreateDataSourceParams{
		ProjectID:   projectID,
		Name:        ds.GetName(),
		DisplayName: ds.GetName(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create data source: %w", err)
	}

	// Create function records based on driver type
	switch drv := ds.GetDriver().(type) {
	case *minderv1.DataSource_Rest:
		for name, def := range drv.Rest.GetDef() {
			defBytes, err := protojson.Marshal(def)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal REST definition: %w", err)
			}

			if _, err := tx.AddDataSourceFunction(ctx, db.AddDataSourceFunctionParams{
				DataSourceID: dsRecord.ID,
				ProjectID:    projectID,
				Name:         name,
				Type:         v1datasources.DataSourceDriverRest,
				Definition:   defBytes,
			}); err != nil {
				return nil, fmt.Errorf("failed to create data source function: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported data source driver type: %T", drv)
	}

	if err := stx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ds, nil
}

// nolint:revive // there is a TODO
func (d *dataSourceService) Update(
	ctx context.Context, ds *minderv1.DataSource, opts *Options) (*minderv1.DataSource, error) {
	//TODO implement me
	panic("implement me")
}

// nolint:revive // there is a TODO
func (d *dataSourceService) Delete(
	ctx context.Context, id uuid.UUID, project uuid.UUID, opts *Options) error {
	//TODO implement me
	panic("implement me")
}

// nolint:revive // there is a TODO
func (d *dataSourceService) ValidateRuleTypeReferences(
	ctx context.Context, rt *minderv1.RuleType, opts *Options) error {
	//TODO implement me
	panic("implement me")
}

// BuildDataSourceRegistry bundles up all data sources referenced in the rule type
// into a registry.
//
// Note that this assumes that the rule type has already been validated.
func (d *dataSourceService) BuildDataSourceRegistry(
	ctx context.Context, rt *minderv1.RuleType, opts *Options) (*v1datasources.DataSourceRegistry, error) {
	rawproj := rt.GetContext().GetProject()
	proj, err := uuid.Parse(rawproj)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project UUID: %w", err)
	}

	instantiations := rt.GetDef().GetEval().GetDataSources()
	reg := v1datasources.NewDataSourceRegistry()

	stx, err := d.txBuilder(d, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	//nolint:gosec // we'll log this error later.
	defer stx.Rollback()

	tx := stx.Q()

	projectHierarchy, err := tx.GetParentProjects(ctx, proj)
	if err != nil {
		return nil, fmt.Errorf("failed to get project hierarchy: %w", err)
	}

	for _, ref := range instantiations {
		inst, err := d.instantiateDataSource(ctx, ref, projectHierarchy, tx)
		if err != nil {
			return nil, fmt.Errorf("failed to instantiate data source: %w", err)
		}

		impl, err := datasources.BuildFromProtobuf(inst)
		if err != nil {
			return nil, fmt.Errorf("failed to build data source from protobuf: %w", err)
		}

		if err := reg.RegisterDataSource(inst.GetName(), impl); err != nil {
			return nil, fmt.Errorf("failed to register data source: %w", err)
		}
	}

	return reg, nil
}
