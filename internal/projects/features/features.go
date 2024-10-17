// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package features provides the features checks for the projects
package features

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
)

const (
	privateReposEnabledFlag               = "private_repositories_enabled"
	projectHierarchyOperationsEnabledFlag = "project_hierarchy_operations_enabled"
)

// ProjectAllowsPrivateRepos checks if the project allows private repositories
func ProjectAllowsPrivateRepos(ctx context.Context, store db.Store, projectID uuid.UUID) bool {
	return featureEnabled(ctx, store, projectID, privateReposEnabledFlag)
}

// ProjectAllowsProjectHierarchyOperations checks if the project allows project hierarchy operations
func ProjectAllowsProjectHierarchyOperations(ctx context.Context, store db.Store, projectID uuid.UUID) bool {
	return featureEnabled(ctx, store, projectID, projectHierarchyOperationsEnabledFlag)
}

// Is a simple helper function to check if a feature is enabled for a project.
// This does not check the feature's configuration, if any, just that it's enabled.
func featureEnabled(ctx context.Context, store db.Store, projectID uuid.UUID, featureFlag string) bool {
	// we're throwing away the result because we're really not interested in what the feature
	// sets, just that it's enabled
	_, err := store.GetFeatureInProject(ctx, db.GetFeatureInProjectParams{
		ProjectID: projectID,
		Feature:   featureFlag,
	})
	if errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Debug().
			Str("project_id", projectID.String()).
			Str("feature", featureFlag).
			Msg("feature disabled for project")
		return false
	} else if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error checking feature for project")
		return false
	}

	zerolog.Ctx(ctx).Debug().
		Str("project_id", projectID.String()).
		Str("feature", featureFlag).
		Msg("feature enabled for project")
	return true
}
