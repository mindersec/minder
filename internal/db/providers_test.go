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
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
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
