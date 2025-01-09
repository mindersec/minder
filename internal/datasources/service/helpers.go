// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/mindersec/minder/internal/datasources"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	getByNameQuery = func(ctx context.Context, tx db.ExtendQuerier, projs []uuid.UUID, name string) (db.DataSource, error) {
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
	}
	getByIDQuery = func(ctx context.Context, tx db.ExtendQuerier, projs []uuid.UUID, id uuid.UUID) (db.DataSource, error) {
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
	}
)

func (d *dataSourceService) getDataSourceSomehow(
	ctx context.Context,
	project uuid.UUID,
	opts *ReadOptions,
	theSomehow func(ctx context.Context, qtx db.ExtendQuerier, projs []uuid.UUID) (db.DataSource, error),
) (*minderv1.DataSource, error) {
	stx, err := d.txBuilder(d, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	//nolint:gosec // we'll log this error later.
	defer stx.Rollback()

	tx := stx.Q()

	ds, err := getDataSourceFromDb(ctx, project, opts, tx, theSomehow)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source from DB: %w", err)
	}

	dsfuncs, err := getDataSourceFunctions(ctx, tx, ds)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source functions: %w", err)
	}

	if err := stx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return dataSourceDBToProtobuf(*ds, dsfuncs)
}

func getDataSourceFromDb(
	ctx context.Context,
	project uuid.UUID,
	opts *ReadOptions,
	qtx db.ExtendQuerier,
	dbQuery func(ctx context.Context, qtx db.ExtendQuerier, projs []uuid.UUID) (db.DataSource, error),
) (*db.DataSource, error) {
	var projs []uuid.UUID
	if len(opts.hierarchy) > 0 {
		projs = opts.hierarchy
	} else {
		prjs, err := listRelevantProjects(ctx, qtx, project, opts.canSearchHierarchical())
		if err != nil {
			return nil, fmt.Errorf("failed to list relevant projects: %w", err)
		}

		projs = prjs
	}

	ds, err := dbQuery(ctx, qtx, projs)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source from DB: %w", err)
	}

	return &ds, nil
}

func getDataSourceFunctions(
	ctx context.Context,
	tx db.ExtendQuerier,
	ds *db.DataSource,
) ([]db.DataSourcesFunction, error) {
	dsfuncs, err := tx.ListDataSourceFunctions(ctx, db.ListDataSourceFunctionsParams{
		DataSourceID: ds.ID,
		ProjectID:    ds.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get data source functions: %w", err)
	}

	// NOTE: We currently treat data sources without functions as an error.
	// The only reason for this is that I (Ozz) currently see no use-case for
	// data sources without functions. If we ever have a use-case for this, we
	// should remove this check.
	if len(dsfuncs) == 0 {
		return nil, errors.New("data source has no functions")
	}

	return dsfuncs, nil
}

func (d *dataSourceService) instantiateDataSource(
	ctx context.Context,
	ref *minderv1.DataSourceReference,
	projectHierarchy []uuid.UUID,
	tx db.ExtendQuerier,
) (*minderv1.DataSource, error) {
	// If we end up supporting other ways of referencing a data source, this
	// would be the place to validate them.
	if ref.GetName() == "" {
		return nil, errors.New("data source name is empty")
	}

	ds, err := d.GetByName(ctx, ref.GetName(), uuid.Nil,
		ReadBuilder().withHierarchy(projectHierarchy).WithTransaction(tx))
	if err != nil {
		return nil, fmt.Errorf("failed to get data source by name: %w", err)
	}

	return ds, nil
}

func listRelevantProjects(
	ctx context.Context, tx db.ExtendQuerier, project uuid.UUID, hierarchical bool,
) ([]uuid.UUID, error) {
	if hierarchical {
		projs, err := tx.GetParentProjects(ctx, project)
		if err != nil {
			return nil, err
		}

		return projs, nil
	}

	return []uuid.UUID{project}, nil
}

func validateDataSourceFunctionsUpdate(
	existingDS *db.DataSource, existingFunctions []db.DataSourcesFunction, newDS *minderv1.DataSource,
) error {
	existingDsProto, err := dataSourceDBToProtobuf(*existingDS, existingFunctions)
	if err != nil {
		// If we got here, it means the existing data source is invalid.
		return fmt.Errorf("failed to convert data source to protobuf: %w", err)
	}

	existingImpl, err := datasources.BuildFromProtobuf(existingDsProto)
	if err != nil {
		// If we got here, it means the existing data source is invalid.
		return fmt.Errorf("failed to build data source from protobuf: %w", err)
	}

	updatedImpl, err := datasources.BuildFromProtobuf(newDS)
	if err != nil {
		return fmt.Errorf("failed to build data source from protobuf: %w", err)
	}

	// We can't validate that the function is not being used. So, we
	// prevent folks from deleting functions.
	// Updates are thus limited to adding new functions and updating existing ones.
	newFuncs := updatedImpl.GetFuncs()
	for key, def := range existingImpl.GetFuncs() {
		newFunc, ok := newFuncs[key]
		if !ok {
			return util.UserVisibleError(codes.InvalidArgument,
				"function %s is missing in the update", key)
		}

		// we validate that the schema update is valid
		if err := def.ValidateUpdate(newFunc.GetArgsSchema()); err != nil {
			return util.UserVisibleError(codes.InvalidArgument,
				"function %s update is invalid: %v", key, err)
		}
	}

	return nil
}
