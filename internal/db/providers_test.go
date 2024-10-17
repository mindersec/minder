// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/rand"
)

func createRandomProvider(t *testing.T, projectID uuid.UUID) Provider {
	t.Helper()

	seed := time.Now().UnixNano()

	prov, err := testQueries.CreateProvider(context.Background(), CreateProviderParams{
		Name:       rand.RandomName(seed),
		ProjectID:  projectID,
		Class:      ProviderClassGithub,
		Implements: []ProviderType{ProviderTypeGithub, ProviderTypeGit},
		AuthFlows:  []AuthorizationFlow{AuthorizationFlowUserInput},
		Definition: json.RawMessage("{}"),
	})
	require.NoError(t, err, "Error creating provider")
	require.NotEmpty(t, prov, "Empty provider returned")

	return prov
}

func TestCreateAndDeleteProvider(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)

	getProv, err := testQueries.GetProviderByID(context.Background(), prov.ID)
	require.NoError(t, err, "Error getting provider")
	require.NotEmpty(t, getProv, "Empty provider returned")

	err = testQueries.DeleteProvider(context.Background(), DeleteProviderParams{ID: prov.ID, ProjectID: proj.ID})
	require.NoError(t, err, "Error deleting provider")

	_, err = testQueries.GetProviderByID(context.Background(), prov.ID)
	require.ErrorIs(t, err, sql.ErrNoRows, "Retrieved provider after deletion")
}
