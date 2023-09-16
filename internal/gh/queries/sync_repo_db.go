// // Copyright 2023 Stacklok, Inc
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //	http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

// Package queries contains the database queries for the GitHub integration
package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/mediator/internal/db"
	ghclient "github.com/stacklok/mediator/internal/providers/github"
)

// SyncRepositoriesWithDB syncs the repositories already in the database with the
// repositories returned from GitHub for a given group ID.
// It works by first getting existing repositories from the database, and we then
// check if the repository already exists in the database, and if it does,
// we check if it needs to be updated.
// If it doesn't exist, we create it.
// This function will be called on initial enrollment by the client (medic enroll ...),
// It can then later be called to sync the repositories with the database.
// In time this maybe better suited to a stored procedure.
// Bench marking this function should 0.8sec for an initial sync of 360 new repos.
//
//gocyclo:ignore
func SyncRepositoriesWithDB(ctx context.Context,
	store db.Store,
	result ghclient.RepositoryListResult,
	provider uuid.UUID) error {
	// Get all existing repositories from the database by group ID
	dbRepos, err := store.ListRepositoriesByProvider(ctx, db.ListRepositoriesByProviderParams{
		Provider: provider,
	})
	if err != nil {
		return fmt.Errorf("error retrieving list of repositories: %w", err)
	}

	// Create a map of the current repositories, so that we can check if a
	// repository already exists in the database against the fresh results returned from GitHub
	dbRepoIDs := make(map[int32]bool, len(dbRepos))
	for _, repo := range dbRepos {
		dbRepoIDs[repo.RepoID] = true
	}

	// Iterate over the repositories returned from GitHub
	for _, repo := range result.Repositories {
		// Check if the repository already exists in the database by Repo ID
		existingRepo, err := store.GetRepositoryByRepoID(ctx, int32(*repo.ID))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// The repository doesn't exist in our DB, let's create it
				_, err = store.CreateRepository(ctx, db.CreateRepositoryParams{
					Provider:  provider,
					RepoOwner: string(*repo.Owner.Login),
					RepoName:  string(*repo.Name),
					RepoID:    int32(*repo.ID),
					IsPrivate: bool(*repo.Private), // Needs a value from GraphQL data
					IsFork:    bool(*repo.Fork),
					CloneUrl:  string(*repo.CloneURL),
				})
				if err != nil {
					fmt.Println("failed to create repository for repo ID: with repo Name: ", *repo.ID, *repo.Name)
					return fmt.Errorf("failed to create repository: %w", err)
				}
				// Delete this newly created repository's ID from the map
				delete(dbRepoIDs, int32(*repo.ID))
			} else {
				// If it's any other error, we just fail the synchronization
				return fmt.Errorf("failed during repository synchronization: %w", err)
			}
		} else {
			if existingRepo.Provider != provider {
				fmt.Println("got request to sync repository of different provider. Skipping")
				continue
			}

			// The repository exists, let's check if it needs to be updated.
			if existingRepo.RepoName != string(*repo.Name) ||
				existingRepo.IsFork != bool(*repo.Fork) {
				fmt.Println("updating repository for repo ID: with repo Name: ", *repo.ID, *repo.Name)
				_, err = store.UpdateRepository(ctx, db.UpdateRepositoryParams{
					Provider:  provider,
					RepoOwner: string(*repo.Owner.Login),
					RepoName:  string(*repo.Name),
					RepoID:    int32(*repo.ID),
					IsPrivate: bool(*repo.Private), // Needs a value from GraphQL data
					IsFork:    bool(*repo.Fork),
					ID:        existingRepo.ID,
					CloneUrl:  string(*repo.CloneURL),
				})
				if err != nil {
					return fmt.Errorf("failed to update repository: %w", err)
				}
			}
			// Delete the repository ID from the map
			delete(dbRepoIDs, int32(*repo.ID))
		}
	}

	// Any remaining repositories in dbRepoNames were not in result.Repositories
	// response from GitHub, so we need to delete them from the database
	for repoID := range dbRepoIDs {

		// Get repository by ID and delete it
		repoToDelete, err := store.GetRepositoryByRepoID(ctx, repoID)
		if err != nil {
			return fmt.Errorf("failed to get repository ID to delete: %w", err)
		}

		err = store.DeleteRepository(ctx, repoToDelete.ID)
		if err != nil {
			return fmt.Errorf("failed to delete repository during sync operation: %w", err)
		}
	}

	return nil
}
