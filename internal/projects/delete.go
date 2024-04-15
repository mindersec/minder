//
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

package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/github/service"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// CleanUpUnmanagedProjects deletes a project if it has no role assignments left
func CleanUpUnmanagedProjects(
	ctx context.Context,
	subject string,
	proj uuid.UUID,
	querier db.Querier,
	authzClient authz.Client,
	providerService service.GitHubProviderService, l zerolog.Logger,
) error {
	l = l.With().Str("project", proj.String()).Logger()
	// Given that we've deleted the user from the authorization system,
	// we can now check if there are any role assignments for the project.
	as, err := authzClient.AssignmentsToProject(ctx, proj)
	if err != nil {
		return fmt.Errorf("error getting role assignments for project %v", err)
	}

	if !hasOtherRoleAssignments(as, subject) {
		l.Info().Msg("deleting project")
		if err := DeleteProject(ctx, proj, querier, authzClient, providerService, l); err != nil {
			return fmt.Errorf("error deleting project %v", err)
		}
	} else {
		l.Debug().Msg("project has other role assignments")
	}
	return nil
}

func hasOtherRoleAssignments(as []*v1.RoleAssignment, subject string) bool {
	return slices.ContainsFunc(as, func(a *v1.RoleAssignment) bool {
		return a.Subject != subject
	})
}

// DeleteProject deletes a project and authorization relationships
func DeleteProject(ctx context.Context, proj uuid.UUID, querier db.Querier, authzClient authz.Client,
	providerService service.GitHubProviderService, l zerolog.Logger) error {
	_, err := querier.GetProjectByID(ctx, proj)
	if err != nil {
		// This project has already been deleted. Skip and go to the next one.
		if errors.Is(err, sql.ErrNoRows) {
			l.Debug().Msg("project already deleted")
			return nil
		}
		return fmt.Errorf("error getting project %v", err)
	}

	// delete associated providers and clean up their state (e.g. GitHub installations)
	dbProviders, err := querier.ListProvidersByProjectID(ctx, []uuid.UUID{proj})
	if err != nil {
		l.Error().Err(err).Msg("error getting providers for project")
	}
	for i := range dbProviders {
		if err := providerService.DeleteProvider(ctx, &dbProviders[i]); err != nil {
			l.Error().Err(err).Msg("error deleting provider")
		}
	}

	// no role assignments for this project
	// we can safely delete it.
	l.Debug().Msg("deleting project from database")
	deletions, err := querier.DeleteProject(ctx, proj)
	if err != nil {
		return fmt.Errorf("error deleting project %v", err)
	}

	for _, d := range deletions {
		if d.ParentID.Valid {
			l.Debug().Str("parent_id", d.ParentID.UUID.String()).Msg("orphaning project")
			if err := authzClient.Orphan(ctx, d.ParentID.UUID, d.ID); err != nil {
				return fmt.Errorf("error orphaning project %v", err)
			}
		}
	}

	return nil
}
