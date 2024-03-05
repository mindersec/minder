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

package reminder

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	cursorutil "github.com/stacklok/minder/internal/util/cursor"
)

type listProjectsResponse struct {
	projects []*db.Project
	cursor   string
}

type listProjectsRequest struct {
	cursor string
	limit  int
}

func (r *reminder) listProjects(ctx context.Context, req listProjectsRequest) (
	*listProjectsResponse, error,
) {
	cursor, err := cursorutil.NewProjectCursor(req.cursor)
	if err != nil {
		return nil, err
	}

	limit := sql.NullInt64{
		// Add 1 to the limit to check if there are more results
		Int64: int64(req.limit + 1),
		Valid: req.limit > 0,
	}
	projects, err := r.store.ListProjects(ctx, db.ListProjectsParams{
		ID: uuid.NullUUID{
			UUID:  cursor.Id,
			Valid: cursor.Id != uuid.Nil,
		},
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	var nextCursor *cursorutil.ProjectCursor
	if limit.Valid && int64(len(projects)) == limit.Int64 {
		nextCursor = &cursorutil.ProjectCursor{
			Id: projects[req.limit].ID,
		}

		// remove the (req.limit + 1)th element from the results
		projects = projects[:req.limit]
	}

	results := make([]*db.Project, len(projects))
	for i := range projects {
		results[i] = &projects[i]
	}

	return &listProjectsResponse{
		projects: results,
		cursor:   nextCursor.String(),
	}, nil
}

type listRepoRequest struct {
	projectId uuid.UUID
	provider  string
	cursor    string
	limit     int
}

type listRepoResponse struct {
	results []*db.Repository
	cursor  string
}

func (r *reminder) listRepositories(ctx context.Context, req listRepoRequest) (*listRepoResponse, error) {
	reqRepoCursor, err := cursorutil.NewRepoCursor(req.cursor)
	if err != nil {
		return nil, err
	}

	repoId := sql.NullInt64{
		Valid: reqRepoCursor.ProjectId == req.projectId.String() &&
			reqRepoCursor.Provider == req.provider,
		Int64: reqRepoCursor.RepoId,
	}

	limit := sql.NullInt64{
		Valid: req.limit > 0,
		Int64: int64(req.limit + 1),
	}

	repos, err := r.store.ListRepositoriesByProjectID(ctx, db.ListRepositoriesByProjectIDParams{
		Provider:  req.provider,
		ProjectID: req.projectId,
		RepoID:    repoId,
		Limit:     limit,
	})

	if err != nil {
		return nil, err
	}

	results := make([]*db.Repository, len(repos))
	for i := range repos {
		results[i] = &repos[i]
	}

	var respRepoCursor *cursorutil.RepoCursor
	if limit.Valid && int64(len(repos)) == limit.Int64 {
		respRepoCursor = &cursorutil.RepoCursor{
			ProjectId: req.projectId.String(),
			Provider:  req.provider,
			RepoId:    repos[req.limit].RepoID,
		}

		// remove the (req.limit + 1)th element from the results
		results = results[:req.limit]
	}

	return &listRepoResponse{
		results: results,
		cursor:  respRepoCursor.String(),
	}, nil
}

type listOldestRuleEvaluationsByIdsResponse struct {
	results []repoOldestRuleEvaluation
}

type repoOldestRuleEvaluation struct {
	repoId               uuid.UUID
	oldestRuleEvaluation time.Time
}

func (r *reminder) listOldestRuleEvaluationsByIds(ctx context.Context, repoId []uuid.UUID) (
	*listOldestRuleEvaluationsByIdsResponse, error,
) {
	oldestRuleEval, err := r.store.ListOldestRuleEvaluationsByRepositoryId(ctx, repoId)
	if err != nil {
		return nil, err
	}

	results := make([]repoOldestRuleEvaluation, len(oldestRuleEval))
	for i := range oldestRuleEval {
		results[i] = repoOldestRuleEvaluation{
			repoId:               oldestRuleEval[i].RepositoryID,
			oldestRuleEvaluation: oldestRuleEval[i].OldestLastUpdated,
		}
	}

	return &listOldestRuleEvaluationsByIdsResponse{
		results: results,
	}, nil
}
