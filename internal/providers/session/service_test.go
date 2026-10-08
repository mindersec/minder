package session_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/crypto/algorithms"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers"
	mock_manager "github.com/mindersec/minder/internal/providers/manager/mock"
	"github.com/mindersec/minder/internal/providers/session"
	mock_session "github.com/mindersec/minder/internal/providers/session/mock"
)

func TestCreateProviderFromSessionState(t *testing.T) {
	t.Parallel()

	stateStr := "test-session-state"
	projectID := uuid.New()
	providerName := "test-provider"
	providerClass := db.ProviderClassGithubApp

	encData := &crypto.EncryptedData{
		Algorithm:   algorithms.Aes256Cfb,
		EncodedData: "dummy",
		KeyVersion:  "v1",
	}

	serializedEnc, err := encData.Serialize()
	require.NoError(t, err)

	stateData := db.GetProjectIDBySessionStateRow{
		ProjectID:   projectID,
		Provider:    providerName,
		OwnerFilter: sql.NullString{String: "test-owner", Valid: true},
		ProviderConfig: []byte(`{}`),
	}

	accessTokenParams := db.UpsertAccessTokenParams{
		ProjectID:       projectID,
		Provider:        providerName,
		OwnerFilter:     stateData.OwnerFilter,
		EnrollmentNonce: sql.NullString{String: stateStr, Valid: true},
		EncryptedAccessToken: pqtype.NullRawMessage{
			RawMessage: serializedEnc,
			Valid:      true,
		},
	}

	dummyProvider := &db.Provider{
		ID:        uuid.New(),
		Name:      providerName,
		ProjectID: projectID,
	}

	t.Run("successfully creates provider and access token when provider does not exist", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mock_session.NewMockdbSessionStore(ctrl)
		mockGetter := mock_session.NewMockproviderByNameGetter(ctrl)
		mockManager := mock_manager.NewMockProviderManager(ctrl)

		mockStore.EXPECT().GetProjectIDBySessionState(gomock.Any(), stateStr).Return(stateData, nil)
		
		// Provider does not exist initially
		mockGetter.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(nil, providers.ErrProviderNotFoundBy{})

		// It should attempt to create it
		mockManager.EXPECT().CreateFromConfig(
			gomock.Any(), providerClass, projectID, providerName, stateData.ProviderConfig,
		).Return(dummyProvider, nil)

		// It should insert access token
		mockStore.EXPECT().UpsertAccessToken(gomock.Any(), accessTokenParams).Return(db.ProviderAccessToken{}, nil)

		svc := session.NewProviderSessionService(mockManager, mockGetter, mockStore)
		res, err := svc.CreateProviderFromSessionState(context.Background(), providerClass, encData, stateStr)

		require.NoError(t, err)
		assert.Equal(t, dummyProvider, res)
	})

	t.Run("successfully updates access token when provider already exists", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mock_session.NewMockdbSessionStore(ctrl)
		mockGetter := mock_session.NewMockproviderByNameGetter(ctrl)
		mockManager := mock_manager.NewMockProviderManager(ctrl)

		mockStore.EXPECT().GetProjectIDBySessionState(gomock.Any(), stateStr).Return(stateData, nil)

		// Provider exists
		mockGetter.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(dummyProvider, nil)

		// It should NOT create it
		
		// It should insert access token
		mockStore.EXPECT().UpsertAccessToken(gomock.Any(), accessTokenParams).Return(db.ProviderAccessToken{}, nil)

		svc := session.NewProviderSessionService(mockManager, mockGetter, mockStore)
		res, err := svc.CreateProviderFromSessionState(context.Background(), providerClass, encData, stateStr)

		require.NoError(t, err)
		assert.Equal(t, dummyProvider, res)
	})

	t.Run("returns existing provider when losing creation race", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mock_session.NewMockdbSessionStore(ctrl)
		mockGetter := mock_session.NewMockproviderByNameGetter(ctrl)
		mockManager := mock_manager.NewMockProviderManager(ctrl)

		mockStore.EXPECT().GetProjectIDBySessionState(gomock.Any(), stateStr).Return(stateData, nil)

		// Provider does not exist initially
		mockGetter.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(nil, providers.ErrProviderNotFoundBy{})

		// Loses race during creation
		uniqueViolationErr := &pq.Error{Code: "23505"} // unique_violation
		mockManager.EXPECT().CreateFromConfig(
			gomock.Any(), providerClass, projectID, providerName, stateData.ProviderConfig,
		).Return(nil, uniqueViolationErr)

		// Fetches the newly created provider by the winner of the race
		mockGetter.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(dummyProvider, nil)

		// Does NOT insert access token because it lost the race

		svc := session.NewProviderSessionService(mockManager, mockGetter, mockStore)
		res, err := svc.CreateProviderFromSessionState(context.Background(), providerClass, encData, stateStr)

		require.ErrorIs(t, err, uniqueViolationErr)
		assert.Equal(t, dummyProvider, res)
	})

	t.Run("fails when GetProjectIDBySessionState returns error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mock_session.NewMockdbSessionStore(ctrl)
		mockGetter := mock_session.NewMockproviderByNameGetter(ctrl)
		mockManager := mock_manager.NewMockProviderManager(ctrl)

		expectedErr := sql.ErrNoRows
		mockStore.EXPECT().GetProjectIDBySessionState(gomock.Any(), stateStr).Return(db.GetProjectIDBySessionStateRow{}, expectedErr)

		svc := session.NewProviderSessionService(mockManager, mockGetter, mockStore)
		res, err := svc.CreateProviderFromSessionState(context.Background(), providerClass, encData, stateStr)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error getting state data by session state")
		assert.Nil(t, res)
	})

	t.Run("fails when GetByName returns an unexpected error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mock_session.NewMockdbSessionStore(ctrl)
		mockGetter := mock_session.NewMockproviderByNameGetter(ctrl)
		mockManager := mock_manager.NewMockProviderManager(ctrl)

		mockStore.EXPECT().GetProjectIDBySessionState(gomock.Any(), stateStr).Return(stateData, nil)

		expectedErr := sql.ErrConnDone
		mockGetter.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(nil, expectedErr)

		svc := session.NewProviderSessionService(mockManager, mockGetter, mockStore)
		res, err := svc.CreateProviderFromSessionState(context.Background(), providerClass, encData, stateStr)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error getting provider from DB")
		assert.Nil(t, res)
	})

	t.Run("fails when CreateFromConfig returns an unexpected error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mock_session.NewMockdbSessionStore(ctrl)
		mockGetter := mock_session.NewMockproviderByNameGetter(ctrl)
		mockManager := mock_manager.NewMockProviderManager(ctrl)

		mockStore.EXPECT().GetProjectIDBySessionState(gomock.Any(), stateStr).Return(stateData, nil)
		mockGetter.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(nil, providers.ErrProviderNotFoundBy{})

		expectedErr := status.Error(codes.Internal, "internal create error")
		mockManager.EXPECT().CreateFromConfig(
			gomock.Any(), providerClass, projectID, providerName, stateData.ProviderConfig,
		).Return(nil, expectedErr)

		svc := session.NewProviderSessionService(mockManager, mockGetter, mockStore)
		res, err := svc.CreateProviderFromSessionState(context.Background(), providerClass, encData, stateStr)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error creating provider")
		assert.Nil(t, res)
	})

	t.Run("fails when UpsertAccessToken returns an error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mock_session.NewMockdbSessionStore(ctrl)
		mockGetter := mock_session.NewMockproviderByNameGetter(ctrl)
		mockManager := mock_manager.NewMockProviderManager(ctrl)

		mockStore.EXPECT().GetProjectIDBySessionState(gomock.Any(), stateStr).Return(stateData, nil)
		mockGetter.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(dummyProvider, nil)

		expectedErr := sql.ErrConnDone
		mockStore.EXPECT().UpsertAccessToken(gomock.Any(), accessTokenParams).Return(db.ProviderAccessToken{}, expectedErr)

		svc := session.NewProviderSessionService(mockManager, mockGetter, mockStore)
		res, err := svc.CreateProviderFromSessionState(context.Background(), providerClass, encData, stateStr)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error inserting access token")
		assert.Nil(t, res)
	})
}
