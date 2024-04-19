// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eea

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/repositories"
	"github.com/stacklok/minder/internal/util/ptr"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// various DB access functions for EEA

// GetRepository retrieves a repository from the database
// and converts it to a protobuf
// Returns provider ID alongside the repository
func getRepository(
	ctx context.Context,
	store db.Querier,
	projectID uuid.UUID,
	repoID uuid.UUID,
) (uuid.UUID, *minderv1.Repository, error) {
	dbrepo, err := store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
		ID:        repoID,
		ProjectID: projectID,
	})
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("error getting repository: %w", err)
	}

	return dbrepo.ProviderID, repositories.PBRepositoryFromDB(dbrepo), nil
}

// GetPullRequest retrieves a pull request from the database
// and converts it to a protobuf
// Returns provider ID alongside the repository
func getPullRequest(
	ctx context.Context,
	store db.Querier,
	projectID,
	repoID,
	pullRequestID uuid.UUID,
) (uuid.UUID, *minderv1.PullRequest, error) {
	// Get repository data - we need the owner and name
	dbrepo, err := store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
		ID:        repoID,
		ProjectID: projectID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil, fmt.Errorf("repository not found")
	} else if err != nil {
		return uuid.Nil, nil, fmt.Errorf("cannot read repository: %v", err)
	}

	dbpr, err := store.GetPullRequestByID(ctx, pullRequestID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil, fmt.Errorf("pull request not found")
	} else if err != nil {
		return uuid.Nil, nil, fmt.Errorf("cannot read pull request: %v", err)
	}

	// TODO: Do we need extra columns in the pull request table?
	return dbrepo.ProviderID, &minderv1.PullRequest{
		Context: &minderv1.Context{
			Project:  ptr.Ptr(dbrepo.ProjectID.String()),
			Provider: ptr.Ptr(dbrepo.Provider),
		},
		Number:    dbpr.PrNumber,
		RepoOwner: dbrepo.RepoOwner,
		RepoName:  dbrepo.RepoName,
	}, nil
}
