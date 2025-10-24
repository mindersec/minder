// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/rand"
)

// EntityRepository is a test helper struct that represents a repository entity
// Used in tests after the legacy Repository table was removed
type EntityRepository struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	ProviderID uuid.UUID
	Name       string
	EntityType Entities
}

// createRandomRepository creates a random repository entity for testing
// This replaces the old createRandomRepository that used the legacy repositories table
func createRandomRepository(t *testing.T, projectID uuid.UUID, prov Provider) EntityRepository {
	t.Helper()

	seed := time.Now().UnixNano()
	repoOwner := rand.RandomName(seed)
	repoName := rand.RandomName(seed)
	name := fmt.Sprintf("%s/%s", repoOwner, repoName)

	// Create entity instance for the repository
	ei, err := testQueries.CreateEntityWithID(context.Background(), CreateEntityWithIDParams{
		ID:         uuid.New(),
		Name:       name,
		ProjectID:  projectID,
		ProviderID: prov.ID,
		EntityType: EntitiesRepository,
	})
	require.NoError(t, err)
	require.NotEmpty(t, ei)

	return EntityRepository{
		ID:         ei.ID,
		ProjectID:  ei.ProjectID,
		ProviderID: ei.ProviderID,
		Name:       ei.Name,
		EntityType: ei.EntityType,
	}
}
