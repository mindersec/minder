// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/crypto/algorithms"
)

func TestUpsertProviderAccessToken(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	project := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, project.ID)

	secret := createSecret(t, "abc")
	serialized := serializeSecret(t, secret)

	tok, err := testQueries.UpsertAccessToken(context.Background(), UpsertAccessTokenParams{
		ProjectID:            project.ID,
		Provider:             prov.Name,
		EncryptedAccessToken: serialized,
		OwnerFilter:          sql.NullString{},
	})

	require.NoError(t, err)
	require.NotEmpty(t, tok)
	require.NotEmpty(t, tok.ID)
	require.NotEmpty(t, tok.CreatedAt)
	require.NotEmpty(t, tok.UpdatedAt)
	require.Equal(t, project.ID, tok.ProjectID)
	require.Equal(t, prov.Name, tok.Provider)
	require.Equal(t, secret, deserializeSecret(t, tok.EncryptedAccessToken))
	require.Equal(t, sql.NullString{}, tok.OwnerFilter)

	newSecret := createSecret(t, "def")
	newSerialized := serializeSecret(t, newSecret)
	tokUpdate, err := testQueries.UpsertAccessToken(context.Background(), UpsertAccessTokenParams{
		ProjectID:            project.ID,
		Provider:             prov.Name,
		EncryptedAccessToken: newSerialized,
		OwnerFilter:          sql.NullString{},
	})

	require.NoError(t, err)
	require.Equal(t, project.ID, tokUpdate.ProjectID)
	require.Equal(t, prov.Name, tokUpdate.Provider)
	require.Equal(t, newSecret, deserializeSecret(t, tokUpdate.EncryptedAccessToken))
	require.Equal(t, sql.NullString{}, tokUpdate.OwnerFilter)
	require.Equal(t, tok.ID, tokUpdate.ID)
	require.Equal(t, tok.CreatedAt, tokUpdate.CreatedAt)
	require.NotEqual(t, tok.UpdatedAt, tokUpdate.UpdatedAt)
}

func createSecret(t *testing.T, encryptedData string) crypto.EncryptedData {
	t.Helper()

	return crypto.EncryptedData{
		Algorithm:   algorithms.Aes256Cfb,
		EncodedData: encryptedData,
		KeyVersion:  "12345",
	}
}

func serializeSecret(t *testing.T, data crypto.EncryptedData) pqtype.NullRawMessage {
	t.Helper()

	serialized, err := data.Serialize()
	require.NoError(t, err)
	return pqtype.NullRawMessage{
		RawMessage: serialized,
		Valid:      true,
	}
}

func deserializeSecret(t *testing.T, data pqtype.NullRawMessage) crypto.EncryptedData {
	t.Helper()

	result, err := crypto.DeserializeEncryptedData(data.RawMessage)
	require.NoError(t, err)
	return result
}
