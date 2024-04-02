//
// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package providers

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	mockcrypto "github.com/stacklok/minder/internal/crypto/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/db/embedded"
	"github.com/stacklok/minder/internal/providers/github/app"
	mockgh "github.com/stacklok/minder/internal/providers/github/mock"
	ghclient "github.com/stacklok/minder/internal/providers/github/oauth"
	mockratecache "github.com/stacklok/minder/internal/providers/ratecache/mock"
	"github.com/stacklok/minder/internal/providers/telemetry"
)

type testMocks struct {
	svcMock     *mockgh.MockClientService
	cryptoMocks *mockcrypto.MockEngine
	fakeStore   db.Store
	cancelFunc  embedded.CancelFunc
}

func testNewProviderService(
	t *testing.T,
	mockCtrl *gomock.Controller,
	config *server.ProviderConfig,
) (*providerService, *testMocks) {
	t.Helper()

	fakeStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	mocks := &testMocks{
		svcMock:     mockgh.NewMockClientService(mockCtrl),
		fakeStore:   fakeStore,
		cancelFunc:  cancelFunc,
		cryptoMocks: mockcrypto.NewMockEngine(mockCtrl),
	}
	require.NoError(t, err)

	psi := NewProviderService(
		mocks.fakeStore,
		mocks.cryptoMocks,
		metrics.NewNoopMetrics(),
		telemetry.NewNoopMetrics(),
		config,
		mockratecache.NewMockRestClientCache(mockCtrl),
	)

	ps, ok := psi.(*providerService)
	require.True(t, ok)
	ps.ghClientService = mocks.svcMock
	return ps, mocks
}

func testCreatePrivateKeyFile(t *testing.T) *os.File {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "pvtkey")
	require.NoError(t, err)

	pvtKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)

	err = pem.Encode(tmpFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(pvtKey),
	})
	require.NoError(t, err)

	return tmpFile
}

func TestProviderService_CreateGitHubOAuthProvider(t *testing.T) {
	t.Parallel()

	const (
		stateNonce       = "test-oauth-nonce"
		stateNonceUpdate = "test-oauth-nonce-update"
	)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &server.ProviderConfig{}

	provSvc, mocks := testNewProviderService(t, ctrl, cfg)
	dbproj, err := mocks.fakeStore.CreateProject(context.Background(),
		db.CreateProjectParams{
			Name:     "test",
			Metadata: []byte(`{}`),
		})
	require.NoError(t, err)

	mocks.cryptoMocks.EXPECT().
		EncryptOAuthToken(gomock.Any()).
		Return([]byte("my-encrypted-token"), nil)

	dbProv, err := provSvc.CreateGitHubOAuthProvider(
		context.Background(),
		ghclient.Github,
		db.ProviderClassGithub,
		oauth2.Token{
			AccessToken: "my-access",
			TokenType:   "Bearer",
		},
		db.GetProjectIDBySessionStateRow{
			ProjectID: dbproj.ID,
			OwnerFilter: sql.NullString{
				Valid:  true,
				String: "testorg",
			},
			RemoteUser: sql.NullString{
				Valid: false,
			},
		},
		stateNonce)
	require.NoError(t, err)
	require.NotNil(t, dbProv)
	require.Equal(t, dbProv.ProjectID, dbproj.ID)
	require.Equal(t, dbProv.AuthFlows, ghclient.AuthorizationFlows)
	require.Equal(t, dbProv.Implements, ghclient.Implements)

	dbToken, err := mocks.fakeStore.GetAccessTokenByProvider(context.Background(), dbProv.Name)
	require.NoError(t, err)
	require.Len(t, dbToken, 1)
	require.Equal(t, dbToken[0].EncryptedToken, base64.StdEncoding.EncodeToString([]byte("my-encrypted-token")))
	require.Equal(t, dbToken[0].OwnerFilter, sql.NullString{String: "testorg", Valid: true})
	require.Equal(t, dbToken[0].EnrollmentNonce, sql.NullString{String: stateNonce, Valid: true})

	// test updating token
	mocks.cryptoMocks.EXPECT().
		EncryptOAuthToken(gomock.Any()).
		Return([]byte("my-new-encrypted-token"), nil)

	dbProvUpdated, err := provSvc.CreateGitHubOAuthProvider(
		context.Background(),
		ghclient.Github,
		db.ProviderClassGithub,
		oauth2.Token{
			AccessToken: "my-access2",
			TokenType:   "Bearer",
		},
		db.GetProjectIDBySessionStateRow{
			ProjectID: dbproj.ID,
			OwnerFilter: sql.NullString{
				Valid:  true,
				String: "testorg",
			},
			RemoteUser: sql.NullString{
				Valid: false,
			},
		},
		stateNonceUpdate)
	require.NoError(t, err)
	require.NotNil(t, dbProv)
	require.Equal(t, dbProvUpdated.ProjectID, dbProv.ProjectID)
	require.Equal(t, dbProvUpdated.AuthFlows, dbProv.AuthFlows)
	require.Equal(t, dbProvUpdated.Implements, dbProv.Implements)

	dbTokenUpdate, err := mocks.fakeStore.GetAccessTokenByProvider(context.Background(), dbProv.Name)
	require.NoError(t, err)
	require.Len(t, dbTokenUpdate, 1)
	require.Equal(t, dbTokenUpdate[0].EncryptedToken, base64.StdEncoding.EncodeToString([]byte("my-new-encrypted-token")))
	require.Equal(t, dbTokenUpdate[0].OwnerFilter, sql.NullString{String: "testorg", Valid: true})
	require.Equal(t, dbTokenUpdate[0].EnrollmentNonce, sql.NullString{String: stateNonceUpdate, Valid: true})
}

func TestProviderService_CreateGitHubAppProvider(t *testing.T) {
	t.Parallel()

	const (
		installationID = 123
		accountLogin   = "test-user"
		accountID      = 456
		stateNonce     = "test-githubapp-nonce"
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pvtKeyFile := testCreatePrivateKeyFile(t)
	defer os.Remove(pvtKeyFile.Name())
	cfg := &server.ProviderConfig{
		GitHubApp: &server.GitHubAppConfig{
			PrivateKey: pvtKeyFile.Name(),
		},
	}

	provSvc, mocks := testNewProviderService(t, ctrl, cfg)
	dbproj, err := mocks.fakeStore.CreateProject(context.Background(),
		db.CreateProjectParams{
			Name:     "test",
			Metadata: []byte(`{}`),
		})
	require.NoError(t, err)

	mocks.svcMock.EXPECT().
		GetInstallation(gomock.Any(), int64(installationID), gomock.Any()).
		Return(&github.Installation{
			Account: &github.User{
				Login: github.String(accountLogin),
				ID:    github.Int64(accountID),
			},
		}, nil, nil)

	dbProv, err := provSvc.CreateGitHubAppProvider(
		context.Background(), oauth2.Token{},
		db.GetProjectIDBySessionStateRow{
			ProjectID: dbproj.ID,
			RemoteUser: sql.NullString{
				Valid: false,
			},
		},
		installationID,
		stateNonce)
	require.NoError(t, err)
	require.NotNil(t, dbProv)

	require.Equal(t, dbProv.ProjectID, dbproj.ID)
	require.Equal(t, dbProv.AuthFlows, app.AuthorizationFlows)
	require.Equal(t, dbProv.Implements, app.Implements)
	require.Equal(t, dbProv.Class, db.NullProviderClass{ProviderClass: db.ProviderClassGithubApp, Valid: true})
	require.Contains(t, dbProv.Name, db.ProviderClassGithubApp)
	require.Contains(t, dbProv.Name, accountLogin)

	dbInstall, err := mocks.fakeStore.GetInstallationIDByProviderID(context.Background(),
		uuid.NullUUID{UUID: dbProv.ID, Valid: true},
	)
	require.NoError(t, err)
	require.Equal(t, dbInstall.AppInstallationID, strconv.FormatInt(installationID, 10))
	require.Equal(t, dbInstall.OrganizationID, int64(accountID))
	require.Equal(t, dbInstall.EnrollmentNonce, sql.NullString{Valid: true, String: stateNonce})

}

func TestProviderService_CreateUnclaimedGitHubAppInstallation(t *testing.T) {
	t.Parallel()

	const (
		installationID = 1234
		accountLogin   = "test-user-unclaimed"
		accountID      = 4567
	)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pvtKeyFile := testCreatePrivateKeyFile(t)
	defer os.Remove(pvtKeyFile.Name())
	cfg := &server.ProviderConfig{
		GitHubApp: &server.GitHubAppConfig{
			PrivateKey: pvtKeyFile.Name(),
		},
	}

	provSvc, mocks := testNewProviderService(t, ctrl, cfg)

	mocks.svcMock.EXPECT().
		GetInstallation(gomock.Any(), int64(installationID), gomock.Any()).
		Return(&github.Installation{
			Account: &github.User{
				Login: github.String(accountLogin),
				ID:    github.Int64(accountID),
			},
		}, nil, nil)

	mocks.svcMock.EXPECT().
		GetUserIdFromToken(gomock.Any(), gomock.Any()).
		Return(github.Int64(accountID), nil)

	dbUnclaimed, err := provSvc.CreateUnclaimedGitHubAppInstallation(
		context.Background(), &oauth2.Token{},
		installationID)
	require.NoError(t, err)
	require.NotNil(t, dbUnclaimed)

	require.Equal(t, dbUnclaimed.ProviderID, uuid.NullUUID{})
	require.Equal(t, dbUnclaimed.AppInstallationID, strconv.FormatInt(installationID, 10))
	require.Equal(t, dbUnclaimed.OrganizationID, int64(accountID))
	require.Equal(t, dbUnclaimed.EnrollingUserID, sql.NullString{Valid: true, String: strconv.FormatInt(accountID, 10)})
}

func TestProviderService_ValidateGithubInstallationId(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pvtKeyFile := testCreatePrivateKeyFile(t)
	defer os.Remove(pvtKeyFile.Name())
	cfg := &server.ProviderConfig{
		GitHubApp: &server.GitHubAppConfig{
			PrivateKey: pvtKeyFile.Name(),
		},
	}

	provSvc, mocks := testNewProviderService(t, ctrl, cfg)

	mocks.svcMock.EXPECT().
		ListUserInstallations(gomock.Any(), gomock.Any()).
		Return([]*github.Installation{
			{
				ID: github.Int64(123),
			},
		}, nil)

	err := provSvc.ValidateGitHubInstallationId(
		context.Background(),
		&oauth2.Token{},
		123)
	require.NoError(t, err)
}
