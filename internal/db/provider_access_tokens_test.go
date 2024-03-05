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

	"github.com/stretchr/testify/require"
)

func TestUpsertProviderAccessToken(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	project := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, project.ID)

	tok, err := testQueries.UpsertAccessToken(context.Background(), UpsertAccessTokenParams{
		ProjectID:      project.ID,
		Provider:       prov.Name,
		EncryptedToken: "abc",
		OwnerFilter:    sql.NullString{},
	})

	require.NoError(t, err)
	require.NotEmpty(t, tok)
	require.NotEmpty(t, tok.ID)
	require.NotEmpty(t, tok.CreatedAt)
	require.NotEmpty(t, tok.UpdatedAt)
	require.Equal(t, project.ID, tok.ProjectID)
	require.Equal(t, prov.Name, tok.Provider)
	require.Equal(t, "abc", tok.EncryptedToken)
	require.Equal(t, sql.NullString{}, tok.OwnerFilter)

	tokUpdate, err := testQueries.UpsertAccessToken(context.Background(), UpsertAccessTokenParams{
		ProjectID:      project.ID,
		Provider:       prov.Name,
		EncryptedToken: "def",
		OwnerFilter:    sql.NullString{},
	})

	require.NoError(t, err)
	require.Equal(t, project.ID, tokUpdate.ProjectID)
	require.Equal(t, prov.Name, tokUpdate.Provider)
	require.Equal(t, "def", tokUpdate.EncryptedToken)
	require.Equal(t, sql.NullString{}, tokUpdate.OwnerFilter)
	require.Equal(t, tok.ID, tokUpdate.ID)
	require.Equal(t, tok.CreatedAt, tokUpdate.CreatedAt)
	require.NotEqual(t, tok.UpdatedAt, tokUpdate.UpdatedAt)
}
