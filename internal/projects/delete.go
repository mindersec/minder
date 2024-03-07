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

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/db"
)

// CleanUpUnmanagedProjects deletes a project if it has no role assignments left
func CleanUpUnmanagedProjects(ctx context.Context, proj uuid.UUID, querier db.Querier, authzClient authz.Client) error {
	// Given that we've deleted the user from the authorization system,
	// we can now check if there are any role assignments for the project.
	as, err := authzClient.AssignmentsToProject(ctx, proj)
	if err != nil {
		return fmt.Errorf("error getting role assignments for project %v", err)
	}

	if len(as) == 0 {
		if err := DeleteProject(ctx, proj, querier, authzClient); err != nil {
			return fmt.Errorf("error deleting project %v", err)
		}
	}
	return nil
}

// DeleteProject deletes a project and authorization relationships
func DeleteProject(ctx context.Context, proj uuid.UUID, querier db.Querier, authzClient authz.Client) error {
	_, err := querier.GetProjectByID(ctx, proj)
	if err != nil {
		// This project has already been deleted. Skip and go to the next one.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("error getting project %v", err)
	}

	// no role assignments for this project
	// we can safely delete it.
	deletions, err := querier.DeleteProject(ctx, proj)
	if err != nil {
		return fmt.Errorf("error deleting project %v", err)
	}

	for _, d := range deletions {
		if d.ParentID.Valid {
			if err := authzClient.Orphan(ctx, d.ParentID.UUID, d.ID); err != nil {
				return fmt.Errorf("error orphaning project %v", err)
			}
		}
	}

	return nil
}
