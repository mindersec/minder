// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/rand"
)

func createRandomRepository(t *testing.T, project uuid.UUID, prov Provider) Repository {
	t.Helper()

	seed := time.Now().UnixNano()
	arg := CreateRepositoryParams{
		Provider:   prov.Name,
		ProviderID: prov.ID,
		ProjectID:  project,
		RepoOwner:  rand.RandomName(seed),
		RepoName:   rand.RandomName(seed),
		RepoID:     rand.RandomInt(0, 5000, seed),
		IsPrivate:  false,
		IsFork:     false,
		WebhookID:  sql.NullInt64{Int64: rand.RandomInt(0, 1000, seed), Valid: true},
		WebhookUrl: randomURL(seed),
		DeployUrl:  randomURL(seed),
	}

	// Generate a unique repo ID by using a larger range
	// to avoid unique constraint violations
	arg.RepoID = rand.RandomInt(10000, 1000000, seed)

	repo, err := testQueries.CreateRepository(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, repo)

	ei, err := testQueries.CreateEntityWithID(context.Background(), CreateEntityWithIDParams{
		ID:         repo.ID,
		Name:       fmt.Sprintf("%s/%s", arg.RepoOwner, arg.RepoName),
		ProjectID:  project,
		ProviderID: prov.ID,
		EntityType: EntitiesRepository,
	})
	require.NoError(t, err)
	require.NotEmpty(t, ei)

	require.Equal(t, arg.Provider, repo.Provider)
	require.Equal(t, arg.ProjectID, repo.ProjectID)
	require.Equal(t, arg.RepoOwner, repo.RepoOwner)
	require.Equal(t, arg.RepoName, repo.RepoName)
	require.Equal(t, arg.RepoID, repo.RepoID)
	require.Equal(t, arg.IsPrivate, repo.IsPrivate)
	require.Equal(t, arg.IsFork, repo.IsFork)
	require.Equal(t, arg.WebhookID, repo.WebhookID)
	require.Equal(t, arg.WebhookUrl, repo.WebhookUrl)

	require.NotZero(t, repo.ID)
	require.NotZero(t, repo.ProjectID)
	require.NotZero(t, repo.CreatedAt)
	require.NotZero(t, repo.UpdatedAt)

	return repo
}

func TestRepository(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	project := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, project.ID)
	createRandomRepository(t, project.ID, prov)
}

// The following tests have been removed as they tested legacy SQL queries
// that have been migrated to entity-based queries. The functionality is now
// tested at a higher level in the repository service tests.

func randomURL(seed int64) string {
	return "http://" + rand.RandomString(10, seed) + ".com"
}
