//
// Copyright 2023 Stacklok, Inc.
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

package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
)

func createRandomPullRequest(t *testing.T, repo uuid.UUID) PullRequest {
	t.Helper()

	seed := time.Now().UnixNano()
	prNum := rand.RandomInt(0, 1000, seed)
	// TODO: we will need an external ID here once the column is set to NOT NULL
	pr, err := testQueries.UpsertPullRequest(context.Background(), UpsertPullRequestParams{
		RepositoryID: repo,
		PrNumber:     prNum,
	})
	require.NoError(t, err)
	require.NotEmpty(t, pr)

	require.Equal(t, repo, pr.RepositoryID)
	require.Equal(t, prNum, pr.PrNumber)

	require.NotZero(t, pr.ID)
	require.NotZero(t, pr.CreatedAt)
	require.NotZero(t, pr.UpdatedAt)

	return pr
}

func TestCreatePullRequest(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	project := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, project.ID)
	repo := createRandomRepository(t, project.ID, prov)

	pr := createRandomPullRequest(t, repo.ID)
	require.NotEmpty(t, pr)

	dbPr, err := testQueries.GetPullRequest(context.Background(), GetPullRequestParams{
		RepositoryID: repo.ID,
		PrNumber:     pr.PrNumber,
	})
	require.NoError(t, err)
	require.NotEmpty(t, dbPr)

	require.Equal(t, pr.ID, dbPr.ID)
	require.Equal(t, pr.RepositoryID, dbPr.RepositoryID)
}

func TestUpsertPullRequest(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	project := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, project.ID)
	repo := createRandomRepository(t, project.ID, prov)

	pr1 := createRandomPullRequest(t, repo.ID)

	arg := UpsertPullRequestParams{
		RepositoryID: repo.ID,
		PrNumber:     pr1.PrNumber,
	}
	pr2, err := testQueries.UpsertPullRequest(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, pr2)

	require.Equal(t, pr1.ID, pr2.ID)
	require.Equal(t, pr1.RepositoryID, pr2.RepositoryID)
	require.Equal(t, pr1.PrNumber, pr2.PrNumber)
	require.Equal(t, pr1.CreatedAt, pr2.CreatedAt)
	require.NotEqual(t, pr1.UpdatedAt, pr2.UpdatedAt)
}

func TestDeletePullRequest(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	project := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, project.ID)
	repo := createRandomRepository(t, project.ID, prov)

	pr1 := createRandomPullRequest(t, repo.ID)
	require.NotEmpty(t, pr1)

	argDel := DeletePullRequestParams{
		RepositoryID: repo.ID,
		PrNumber:     pr1.PrNumber,
	}
	err := testQueries.DeletePullRequest(context.Background(), argDel)
	require.NoError(t, err)

	dbPr, err := testQueries.GetPullRequest(context.Background(), GetPullRequestParams{
		RepositoryID: repo.ID,
		PrNumber:     pr1.PrNumber,
	})
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, dbPr)

	// test that upserting the same PR number creates a new PR
	argUp := UpsertPullRequestParams{
		RepositoryID: repo.ID,
		PrNumber:     pr1.PrNumber,
	}
	pr2, err := testQueries.UpsertPullRequest(context.Background(), argUp)
	require.NoError(t, err)
	require.NotEmpty(t, pr2)
	require.NotEqual(t, pr1.CreatedAt, pr2.CreatedAt)
}
