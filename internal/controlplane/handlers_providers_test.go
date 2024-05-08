// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/authz/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	mockcrypto "github.com/stacklok/minder/internal/crypto/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/providers"
	ghmanager "github.com/stacklok/minder/internal/providers/github/manager"
	mockgh "github.com/stacklok/minder/internal/providers/github/mock"
	mockprovsvc "github.com/stacklok/minder/internal/providers/github/service/mock"
	"github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/providers/ratecache"
	mockwebhooks "github.com/stacklok/minder/internal/repositories/github/webhooks/mock"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestDeleteProvider(t *testing.T) {
	t.Parallel()

	user := openid.New()
	assert.NoError(t, user.Set("sub", "testuser"))

	authzClient := &mock.SimpleClient{
		Allowed: []uuid.UUID{uuid.New()},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	providerName := "test-provider"
	providerID := uuid.New()
	projectID := uuid.New()
	projectIDStr := projectID.String()
	accessToken := "test-token"

	mockProvidersSvc := mockprovsvc.NewMockGitHubProviderService(ctrl)
	mockProvidersSvc.EXPECT().DeleteInstallation(gomock.Any(), gomock.Any()).Return(nil)

	mockCryptoEngine := mockcrypto.NewMockEngine(ctrl)
	mockCryptoEngine.EXPECT().
		DecryptOAuthToken(gomock.Any()).
		Return(oauth2.Token{AccessToken: accessToken}, nil).AnyTimes()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetProviderByName(gomock.Any(), gomock.Any()).Return(db.Provider{
		Name:      providerName,
		ProjectID: projectID,
		Implements: []db.ProviderType{
			db.ProviderTypeGithub,
		},
		ID:         providerID,
		Version:    provinfv1.V1,
		Definition: json.RawMessage(`{"github-app": {}}`),
		Class:      db.ProviderClassGithubApp,
	}, nil)
	mockStore.EXPECT().
		GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
		Return(db.ProviderAccessToken{
			EncryptedToken: "encryptedToken",
		}, nil).AnyTimes()
	mockStore.EXPECT().DeleteProvider(gomock.Any(), db.DeleteProviderParams{
		ID:        providerID,
		ProjectID: projectID,
	}).Return(nil)
	mockStore.EXPECT().
		GetProviderWebhooks(gomock.Any(), gomock.Eq(providerID)).
		Return([]db.GetProviderWebhooksRow{
			{
				RepoOwner: "test",
				RepoName:  "repo",
				WebhookID: sql.NullInt64{Valid: true, Int64: 12345},
			}}, nil)

	whManager := mockwebhooks.NewMockWebhookManager(ctrl)
	whManager.EXPECT().DeleteWebhook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	cancelable, cancel := context.WithCancel(context.Background())
	clientCache := ratecache.NewRestClientCache(cancelable)
	defer cancel()

	gh := mockgh.NewMockGitHub(ctrl)

	clientCache.Set("", accessToken, db.ProviderTypeGithub, gh)

	// I am not replacing provider store with a stub since I want to reuse
	// these tests to test the logic in GitHubProviderClassManager
	providerStore := providers.NewProviderStore(mockStore)
	githubProviderManager := ghmanager.NewGitHubProviderClassManager(
		clientCache,
		nil,
		&serverconfig.ProviderConfig{},
		nil,
		mockCryptoEngine,
		whManager,
		mockStore,
		mockProvidersSvc,
	)
	providerManager, err := manager.NewProviderManager(providerStore, githubProviderManager)
	require.NoError(t, err)

	server := Server{
		cryptoEngine:    mockCryptoEngine,
		store:           mockStore,
		ghProviders:     mockProvidersSvc,
		authzClient:     authzClient,
		providerStore:   providerStore,
		providerManager: providerManager,
		cfg:             &serverconfig.Config{},
	}

	ctx := context.Background()
	ctx = auth.WithAuthTokenContext(ctx, user)
	ctx = engine.WithEntityContext(ctx, &engine.EntityContext{
		Project:  engine.Project{ID: projectID},
		Provider: engine.Provider{Name: providerName},
	})

	resp, err := server.DeleteProvider(ctx, &minder.DeleteProviderRequest{
		Context: &minder.Context{
			Project:  &projectIDStr,
			Provider: &providerName,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, providerName, resp.Name)
}

func TestDeleteProviderByID(t *testing.T) {
	t.Parallel()

	user := openid.New()
	assert.NoError(t, user.Set("sub", "testuser"))

	authzClient := &mock.SimpleClient{
		Allowed: []uuid.UUID{uuid.New()},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	providerName := "test-provider"
	providerID := uuid.New()
	projectID := uuid.New()
	projectIDStr := projectID.String()
	accessToken := "test-token"

	mockProvidersSvc := mockprovsvc.NewMockGitHubProviderService(ctrl)
	mockProvidersSvc.EXPECT().DeleteInstallation(gomock.Any(), gomock.Any()).Return(nil)

	mockCryptoEngine := mockcrypto.NewMockEngine(ctrl)
	mockCryptoEngine.EXPECT().
		DecryptOAuthToken(gomock.Any()).
		Return(oauth2.Token{AccessToken: accessToken}, nil).AnyTimes()

	mockStore := mockdb.NewMockStore(ctrl)
	params := db.GetProviderByIDAndProjectParams{
		ID:        providerID,
		ProjectID: projectID,
	}
	mockStore.EXPECT().GetProviderByIDAndProject(gomock.Any(), params).Return(db.Provider{
		Name:      providerName,
		ID:        providerID,
		ProjectID: projectID,
		Implements: []db.ProviderType{
			db.ProviderTypeGithub,
		},
		Version:    provinfv1.V1,
		Definition: json.RawMessage(`{"github-app": {}}`),
		Class:      db.ProviderClassGithubApp,
	}, nil)
	mockStore.EXPECT().DeleteProvider(gomock.Any(), db.DeleteProviderParams{
		ID:        providerID,
		ProjectID: projectID,
	}).Return(nil)
	mockStore.EXPECT().
		GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
		Return(db.ProviderAccessToken{
			EncryptedToken: "encryptedToken",
		}, nil).AnyTimes()
	mockStore.EXPECT().
		GetProviderWebhooks(gomock.Any(), gomock.Eq(providerID)).
		Return([]db.GetProviderWebhooksRow{
			{
				RepoOwner: "test",
				RepoName:  "repo",
				WebhookID: sql.NullInt64{Valid: true, Int64: 12345},
			}}, nil)

	whManager := mockwebhooks.NewMockWebhookManager(ctrl)
	whManager.EXPECT().DeleteWebhook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	cancelable, cancel := context.WithCancel(context.Background())
	clientCache := ratecache.NewRestClientCache(cancelable)
	defer cancel()

	gh := mockgh.NewMockGitHub(ctrl)

	clientCache.Set("", accessToken, db.ProviderTypeGithub, gh)

	providerStore := providers.NewProviderStore(mockStore)
	githubProviderManager := ghmanager.NewGitHubProviderClassManager(
		clientCache,
		nil,
		&serverconfig.ProviderConfig{},
		nil,
		mockCryptoEngine,
		whManager,
		mockStore,
		mockProvidersSvc,
	)
	providerManager, err := manager.NewProviderManager(providerStore, githubProviderManager)
	require.NoError(t, err)

	server := Server{
		cryptoEngine:    mockCryptoEngine,
		store:           mockStore,
		ghProviders:     mockProvidersSvc,
		authzClient:     authzClient,
		providerStore:   providerStore,
		providerManager: providerManager,
		cfg:             &serverconfig.Config{},
	}

	ctx := context.Background()
	ctx = auth.WithAuthTokenContext(ctx, user)
	ctx = engine.WithEntityContext(ctx, &engine.EntityContext{
		Project: engine.Project{ID: projectID},
	})

	resp, err := server.DeleteProviderByID(ctx, &minder.DeleteProviderByIDRequest{
		Context: &minder.Context{
			Project: &projectIDStr,
		},
		Id: providerID.String(),
	})
	assert.NoError(t, err)
	assert.Equal(t, providerID.String(), resp.Id)
}
