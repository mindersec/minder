// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package session_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers"
	mock_manager "github.com/mindersec/minder/internal/providers/manager/mock"
	"github.com/mindersec/minder/internal/providers/session"
	mock_session "github.com/mindersec/minder/internal/providers/session/mock"
)

type testEnv struct {
	providerManager *mock_manager.MockProviderManager
	provGetter      *mock_session.MockproviderByNameGetter
	dbStore         *mock_session.MockdbSessionStore
	service         session.ProviderSessionService
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctrl := gomock.NewController(t)

	provMgr := mock_manager.NewMockProviderManager(ctrl)
	provGetter := mock_session.NewMockproviderByNameGetter(ctrl)
	dbStore := mock_session.NewMockdbSessionStore(ctrl)

	svc := session.NewProviderSessionService(provMgr, provGetter, dbStore)

	return &testEnv{
		providerManager: provMgr,
		provGetter:      provGetter,
		dbStore:         dbStore,
		service:         svc,
	}
}

func TestCreateProviderFromSessionState(t *testing.T) {
	t.Parallel()

	testProjectID := uuid.New()
	testProviderName := "github-provider"
	testSessionState := "session-state-token-12345"
	testOwnerFilter := sql.NullString{String: "test-org", Valid: true}
	testProviderConfig := json.RawMessage(`{"endpoint":"https://api.github.com"}`)

	validCreds := &crypto.EncryptedData{
		EncodedData: "encrypted-token-data-payload",
		Algorithm:   "aes-256-gcm",
	}

	validSerialized, err := validCreds.Serialize()
	require.NoError(t, err)

	existingProvider := &db.Provider{
		ID:        uuid.New(),
		Name:      testProviderName,
		ProjectID: testProjectID,
		Class:     db.ProviderClassGithub,
	}

	createdProvider := &db.Provider{
		ID:        uuid.New(),
		Name:      testProviderName,
		ProjectID: testProjectID,
		Class:     db.ProviderClassGithub,
	}

	sessionRow := db.GetProjectIDBySessionStateRow{
		ProjectID:      testProjectID,
		Provider:       testProviderName,
		ProviderConfig: testProviderConfig,
		OwnerFilter:    testOwnerFilter,
	}

	expectedUpsertParams := db.UpsertAccessTokenParams{
		ProjectID:       testProjectID,
		Provider:        testProviderName,
		OwnerFilter:     testOwnerFilter,
		EnrollmentNonce: sql.NullString{String: testSessionState, Valid: true},
		EncryptedAccessToken: pqtype.NullRawMessage{
			RawMessage: validSerialized,
			Valid:      true,
		},
	}

	uniqueViolationErr := &pq.Error{
		Code:    pqerror.UniqueViolation,
		Message: "duplicate key value violates unique constraint",
	}

	tests := []struct {
		name                  string
		providerClass         db.ProviderClass
		creds                 *crypto.EncryptedData
		state                 string
		setupMocks            func(env *testEnv)
		expectedProv          *db.Provider
		expectedErr           string
		expectUniqueViolation bool
	}{
		{
			name:          "provider already exists refreshes access token and returns existing provider",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(sessionRow, nil)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(existingProvider, nil)

				env.dbStore.EXPECT().
					UpsertAccessToken(gomock.Any(), expectedUpsertParams).
					Return(db.ProviderAccessToken{}, nil)
			},
			expectedProv: existingProvider,
		},
		{
			name:          "provider does not exist creates new provider and saves access token",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(sessionRow, nil)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(nil, providers.ErrProviderNotFoundBy{})

				env.providerManager.EXPECT().
					CreateFromConfig(gomock.Any(), db.ProviderClassGithub, testProjectID, testProviderName, testProviderConfig).
					Return(createdProvider, nil)

				env.dbStore.EXPECT().
					UpsertAccessToken(gomock.Any(), expectedUpsertParams).
					Return(db.ProviderAccessToken{}, nil)
			},
			expectedProv: createdProvider,
		},
		{
			name:          "fails when session state lookup returns db error",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(db.GetProjectIDBySessionStateRow{}, errors.New("db connection failure"))
			},
			expectedErr: "error getting state data by session state: db connection failure",
		},
		{
			name:          "fails when provider lookup returns unexpected db error",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(sessionRow, nil)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(nil, errors.New("unexpected postgres error"))
			},
			expectedErr: "error getting provider from DB: unexpected postgres error",
		},
		{
			name:          "fails when creating provider returns error",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(sessionRow, nil)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(nil, providers.ErrProviderNotFoundBy{})

				env.providerManager.EXPECT().
					CreateFromConfig(gomock.Any(), db.ProviderClassGithub, testProjectID, testProviderName, testProviderConfig).
					Return(nil, errors.New("invalid provider config schema"))
			},
			expectedErr: "error creating provider: invalid provider config schema",
		},
		{
			name:          "loses creation race on unique constraint returns winner provider and unique violation error",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(sessionRow, nil)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(nil, providers.ErrProviderNotFoundBy{})

				env.providerManager.EXPECT().
					CreateFromConfig(gomock.Any(), db.ProviderClassGithub, testProjectID, testProviderName, testProviderConfig).
					Return(nil, uniqueViolationErr)

				// Fetch existing winner provider
				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(existingProvider, nil)
			},
			expectedProv:          existingProvider,
			expectedErr:           "duplicate key value violates unique constraint",
			expectUniqueViolation: true,
		},
		{
			name:          "loses creation race but fetching winning provider fails returns db error",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(sessionRow, nil)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(nil, providers.ErrProviderNotFoundBy{})

				env.providerManager.EXPECT().
					CreateFromConfig(gomock.Any(), db.ProviderClassGithub, testProjectID, testProviderName, testProviderConfig).
					Return(nil, uniqueViolationErr)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(nil, errors.New("connection reset"))
			},
			expectedErr: "error getting provider from DB after losing creation race: connection reset",
		},
		{
			name:          "fails when access token upsert returns db error",
			providerClass: db.ProviderClassGithub,
			creds:         validCreds,
			state:         testSessionState,
			setupMocks: func(env *testEnv) {
				env.dbStore.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), testSessionState).
					Return(sessionRow, nil)

				env.provGetter.EXPECT().
					GetByName(gomock.Any(), testProjectID, testProviderName).
					Return(existingProvider, nil)

				env.dbStore.EXPECT().
					UpsertAccessToken(gomock.Any(), expectedUpsertParams).
					Return(db.ProviderAccessToken{}, errors.New("disk full"))
			},
			expectedErr: "error inserting access token: disk full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestEnv(t)
			if tt.setupMocks != nil {
				tt.setupMocks(env)
			}

			prov, err := env.service.CreateProviderFromSessionState(
				context.Background(),
				tt.providerClass,
				tt.creds,
				tt.state,
			)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)

				if tt.expectUniqueViolation {
					assert.True(t, db.ErrIsUniqueViolation(err))
				} else {
					assert.False(t, db.ErrIsUniqueViolation(err))
				}

				if tt.expectedProv != nil {
					assert.Equal(t, tt.expectedProv, prov)
				} else {
					assert.Nil(t, prov)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedProv, prov)
		})
	}
}
