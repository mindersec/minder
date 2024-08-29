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
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers/manager"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ProjectDeleter encapsulates operations for deleting projects
// This is a separate interface/struct to ProjectCreator to avoid a circular
// dependency issue.
type ProjectDeleter interface {
	// CleanUpUnmanagedProjects deletes a project if it has no role assignments left
	CleanUpUnmanagedProjects(
		ctx context.Context,
		subject string,
		proj uuid.UUID,
		querier db.Querier,
	) error

	// DeleteProject deletes a project and authorization relationships
	DeleteProject(
		ctx context.Context,
		proj uuid.UUID,
		querier db.Querier,
	) error
}

type projectDeleter struct {
	authzClient     authz.Client
	providerManager manager.ProviderManager
}

// NewProjectDeleter creates a new instance of the project deleter
func NewProjectDeleter(
	authzClient authz.Client,
	providerManager manager.ProviderManager,
) ProjectDeleter {
	return &projectDeleter{
		authzClient:     authzClient,
		providerManager: providerManager,
	}
}

func (p *projectDeleter) CleanUpUnmanagedProjects(
	ctx context.Context,
	subject string,
	proj uuid.UUID,
	querier db.Querier,
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
	as, err := p.authzClient.AssignmentsToProject(ctx, proj)
	if err != nil {
		return fmt.Errorf("error getting role assignments for project: %v", err)
	}

	if !hasOtherRoleAssignments(as, subject) {
		l.Info().Msg("deleting project")
		if err := p.DeleteProject(ctx, proj, querier); err != nil {
			return fmt.Errorf("error deleting project %v", err)
		}
	}
	l.Debug().Msg("project has other administrators, skipping deletion")
	return nil
}

// DeleteProject deletes a project and authorization relationships
func (p *projectDeleter) DeleteProject(
	ctx context.Context,
	proj uuid.UUID,
	querier db.Querier,
) error {
	l := zerolog.Ctx(ctx).With().
		Str("component", "projects").
		Str("operation", "delete").
		Str("project", proj.String()).
		Logger()
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
		if err := p.providerManager.DeleteByID(ctx, provider.ID, proj); err != nil {
			l.Error().Err(err).Msg("error deleting provider")
		}
	}

	projectTombstone, err := exportProjectMetadata(ctx, proj, querier)
	if err != nil {
		return err
	}

	// no role assignments for this project
	// we can safely delete it.
	l.Debug().Msg("deleting project from database")
	deletions, err := querier.DeleteProject(ctx, proj)
	if err != nil {
		return fmt.Errorf("error deleting project %v", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProjectTombstone = *projectTombstone

	for _, d := range deletions {
		if d.ParentID.Valid {
			l.Debug().Str("parent_id", d.ParentID.UUID.String()).Msg("orphaning project")
			if err := p.authzClient.Orphan(ctx, d.ParentID.UUID, d.ID); err != nil {
				return fmt.Errorf("error orphaning project %v", err)
			}
		}
	}

	return nil
}

func hasOtherRoleAssignments(as []*v1.RoleAssignment, subject string) bool {
	return slices.ContainsFunc(as, func(a *v1.RoleAssignment) bool {
		return a.GetRole() == authz.RoleAdmin.String() && a.Subject != subject
	})
}

func exportProjectMetadata(ctx context.Context, projectID uuid.UUID, qtx db.Querier) (*logger.ProjectTombstone, error) {
	var err error

	profilesCount, err := qtx.CountProfilesByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error getting profiles count: %w", err)
	}

	reposCount, err := qtx.CountRepositoriesByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error getting repositories count: %w", err)
	}

	entitlementFeatures, err := qtx.GetEntitlementFeaturesByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error getting entitlement features: %w", err)
	}

	return &logger.ProjectTombstone{
		Project:           projectID,
		ProfileCount:      int(profilesCount),
		RepositoriesCount: int(reposCount),
		Entitlements:      entitlementFeatures,
	}, nil
}
