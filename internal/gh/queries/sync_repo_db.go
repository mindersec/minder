// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package queries contains the database queries for the GitHub integration
package queries

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/go-github/v53/github"
	"github.com/stacklok/mediator/pkg/db"
)

// SyncRepositoriesWithDB syncs the repositories already in the database with the
// repositories returned from GitHub for a given group ID.
// It works by first getting existing repositories from the database, and we then
// check if the repository already exists in the database, and if it does,
// we check if it needs to be updated.
// If it doesn't exist, we create it.
// This function will be called on initial enrollment by the client (medctl enroll ...),
// It can then later be called to sync the repositories with the database.
// In time this maybe better suited to a stored procedure.
// Bench marking this function should 0.8sec for an initial sync of 360 new repos.
//
//gocyclo:ignore
func SyncRepositoriesWithDB(ctx context.Context,
	store db.Store,
	repos []*github.Repository,
	groupId int32) error {
	// Get all existing repositories from the database by group ID
	dbRepos, err := store.ListRepositoriesByGroupID(ctx, db.ListRepositoriesByGroupIDParams{
		GroupID: groupId,
	})
	if err != nil {
		return fmt.Errorf("error retrieving list of repositories: %w", err)
	}

	// Create a map of the current repositories, so that we can check if a
	// repository already exists in the database against the fresh results returned from GitHub
	dbRepoNames := make(map[string]bool)
	for _, repo := range dbRepos {
		dbRepoNames[repo.RepoName] = true
	}

	for _, repo := range repos {
		existingRepo, err := store.GetRepositoryByRepoName(ctx, *repo.Name)

		if err != nil {
			if err == sql.ErrNoRows {
				// The repository doesn't exist in our DB, let's create it
				_, err = store.CreateRepository(ctx, db.CreateRepositoryParams{
					GroupID:   groupId,
					RepoOwner: *repo.Owner.Login,
					RepoName:  *repo.Name,
					RepoID:    int32(*repo.ID),
					IsPrivate: *repo.Private,
					IsFork:    *repo.Fork,
				})
				if err != nil {
					return fmt.Errorf("failed to create repository: %w", err)
				}
			} else {
				// If it's any other error, we just fail the synchronization
				return fmt.Errorf("failed during repository update: %w", err)
			}
		} else {
			// The repository exists, let's check if it needs to be updated.
			if existingRepo.RepoOwner != *repo.Owner.Login ||
				existingRepo.RepoName != *repo.Name ||
				existingRepo.RepoID != int32(*repo.ID) ||
				existingRepo.IsPrivate != *repo.Private ||
				existingRepo.IsFork != *repo.Fork {
				_, err = store.UpdateRepository(ctx, db.UpdateRepositoryParams{
					GroupID:   existingRepo.GroupID,
					RepoOwner: *repo.Owner.Login,
					RepoName:  *repo.Name,
					RepoID:    int32(*repo.ID),
					IsPrivate: *repo.Private,
					IsFork:    *repo.Fork,
					ID:        existingRepo.ID,
				})
				if err != nil {
					return fmt.Errorf("failed to update repository: %w", err)
				}
			}
		}
		// Delete an element with the specified key (m[key]) from the map.
		// If m is nil or there is no such element, delete is a no-op.
		delete(dbRepoNames, *repo.Name)
	}

	// Any remaining repositories in dbRepoNames were not in repos.Repositories
	// response from GitHub, so we need to delete them from the database
	for repoName := range dbRepoNames {

		// Get repository by name (or ID) and delete it
		repoToDelete, err := store.GetRepositoryByRepoName(ctx, repoName)
		if err != nil {
			return fmt.Errorf("failed to get repository name to delete: %w", err)
		}

		err = store.DeleteRepository(ctx, repoToDelete.ID)
		if err != nil {
			return fmt.Errorf("failed to delete repository during sync operation: %w", err)
		}
	}

	return nil
}
