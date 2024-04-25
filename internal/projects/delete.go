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
	"github.com/stacklok/minder/internal/providers/manager"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// CleanUpUnmanagedProjects deletes a project if it has no role assignments left
func CleanUpUnmanagedProjects(
	ctx context.Context,
	subject string,
	proj uuid.UUID,
	querier db.Querier,
	authzClient authz.Client,
	providerManager manager.ProviderManager,
) error {
	l := zerolog.Ctx(ctx).With().Str("project", proj.String()).Logger()
	// We know that non-root projects have a parent which has an admin, so
	// the only projects without management are top-level projects.
	dbProj, err := querier.GetProjectByID(ctx, proj)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// was deleted somehow, log and return okay.
			zerolog.Ctx(ctx).Debug().Str("project_id", proj.String()).Msg("project already deleted")
			return nil
		}
		return err
	}
	if dbProj.ParentID.Valid {
		// This is not a top-level project, skip.
		return nil
	}
	// Given that we've deleted the user assignment from the authorization
	// system, we can now check if there are any remaining role assignments
	// for the project.
	//
	// Note that AssignmentsToProject returns only direct assignments, but
	// we shouldn't have any transitive assignments for root projects.
	// When we add group support, we'll need to check whether admin groups
	// are empty in this check.  The alternative is to use Expand(), but
	// that requires expanding the resulting tree ourselves.
	as, err := authzClient.AssignmentsToProject(ctx, proj)
	if err != nil {
		return fmt.Errorf("error getting role assignments for project: %v", err)
	}

	if !hasOtherRoleAssignments(as, subject) {
		l.Info().Msg("deleting project")
		if err := DeleteProject(ctx, proj, querier, authzClient, providerManager, l); err != nil {
			return fmt.Errorf("error deleting project %v", err)
		}
	}
	l.Debug().Msg("project has other administrators, skipping deletion")
	return nil
}

func hasOtherRoleAssignments(as []*v1.RoleAssignment, subject string) bool {
	return slices.ContainsFunc(as, func(a *v1.RoleAssignment) bool {
		return a.GetRole() == authz.AuthzRoleAdmin.String() && a.Subject != subject
	})
}

// DeleteProject deletes a project and authorization relationships
func DeleteProject(
	ctx context.Context,
	proj uuid.UUID,
	querier db.Querier,
	authzClient authz.Client,
	providerManager manager.ProviderManager,
	l zerolog.Logger,
) error {
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
	for _, provider := range dbProviders {
		if err := providerManager.DeleteByID(ctx, provider.ID, proj); err != nil {
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
