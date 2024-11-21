// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

	projs, err := listRelevantProjects(ctx, tx, project, opts.canSearchHierarchical())
	if err != nil {
		return nil, fmt.Errorf("failed to list relevant projects: %w", err)
	}

	ds, err := theSomehow(ctx, tx, projs)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source by name: %w", err)
	}

	dsfuncs, err := tx.ListDataSourceFunctions(ctx, db.ListDataSourceFunctionsParams{
		DataSourceID: ds.ID,
		ProjectID:    ds.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get data source functions: %w", err)
	}

	if err := stx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return dataSourceDBToProtobuf(ds, dsfuncs)
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
