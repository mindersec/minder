// Copyright 2024 Stacklok, Inc.
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

// Package repositories contains repository logic which is not coupled to any
// single provider (e.g. GitHub)
package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
)

// ProjectAllowsPrivateRepos checks whether a project is configured to allow
// private repos
func ProjectAllowsPrivateRepos(ctx context.Context, store db.Store, projectID uuid.UUID) bool {
	// we're throwing away the result because we're really not interested in what the feature
	// sets, just that it's enabled
	logger := zerolog.Ctx(ctx).
		With().Str("project_id", projectID.String()).
		Logger()

	_, err := store.GetFeatureInProject(ctx, db.GetFeatureInProjectParams{
		ProjectID: projectID,
		Feature:   "private_repositories_enabled",
	})
	if errors.Is(err, sql.ErrNoRows) {
		logger.Debug().
			Msg("private repositories not enabled for project")
		return false
	} else if err != nil {
		logger.Error().
			Msgf("error getting features for project %s: %v", projectID, err)
		return false
	}

	zerolog.Ctx(ctx).Debug().
		Msg("project allows private repositories")
	return true
}
