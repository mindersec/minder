// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/crypto"
	mockcrypto "github.com/mindersec/minder/internal/crypto/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/pkg/config/server"
	mockinterfaces "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestNewGitLabProviderClassManager(t *testing.T) {
	t.Parallel()

	validGitLabCfg := &server.GitLabConfig{
		WebhookSecrets: server.WebhookSecrets{
			WebhookSecret: "test-webhook-secret-12345",
		},
	}

	tests := []struct {
		name        string
		cfg         *server.GitLabConfig
		wgCfg       server.WebhookConfig
		gitCfg      server.GitConfig
		wantErr     bool
		expectedErr string
	}{
		{
			name: "successfully creates manager with valid configurations",
			cfg:  validGitLabCfg,
			wgCfg: server.WebhookConfig{
				ExternalWebhookURL: "https://minder.example.com/api/v1/webhook",
			},
			gitCfg:  server.GitConfig{},
			wantErr: false,
		},
		{
			name: "missing external webhook URL returns error",
			cfg:  validGitLabCfg,
			wgCfg: server.WebhookConfig{
				ExternalWebhookURL: "",
			},
			gitCfg:      server.GitConfig{},
			wantErr:     true,
			expectedErr: "webhook URL is required",
		},
		{
			name: "nil gitlab config returns error",
			cfg:  nil,
			wgCfg: server.WebhookConfig{
				ExternalWebhookURL: "https://minder.example.com/api/v1/webhook",
			},
			gitCfg:      server.GitConfig{},
			wantErr:     true,
			expectedErr: "gitlab config is required",
		},
		{
			name: "missing webhook secret file returns error",
			cfg: &server.GitLabConfig{
				WebhookSecrets: server.WebhookSecrets{
					WebhookSecretFile: "nonexistent-secret-file.txt",
				},
			},
			wgCfg: server.WebhookConfig{
				ExternalWebhookURL: "https://minder.example.com/api/v1/webhook",
			},
			gitCfg:      server.GitConfig{},
			wantErr:     true,
			expectedErr: "error getting webhook secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockStore := mockdb.NewMockStore(ctrl)
			mockCrypt := mockcrypto.NewMockEngine(ctrl)
			mockPub := mockinterfaces.NewMockPublisher(ctrl)

			mgr, err := NewGitLabProviderClassManager(
				context.Background(),
				mockCrypt,
				mockStore,
				mockPub,
				tt.cfg,
				tt.wgCfg,
				tt.gitCfg,
			)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, mgr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, mgr)
				assert.Equal(t, []db.ProviderClass{db.ProviderClassGitlab}, mgr.GetSupportedClasses())
				assert.Equal(t, "https://minder.example.com/api/v1/webhook/gitlab", mgr.webhookURL)
				assert.Equal(t, "test-webhook-secret-12345", mgr.currentWebhookSecret)
			}
		})
	}
}

func TestGetProviderClassInfo(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockStore := mockdb.NewMockStore(ctrl)
	mockCrypt := mockcrypto.NewMockEngine(ctrl)
	mockPub := mockinterfaces.NewMockPublisher(ctrl)

	mgr, err := NewGitLabProviderClassManager(
		context.Background(),
		mockCrypt,
		mockStore,
		mockPub,
		&server.GitLabConfig{WebhookSecrets: server.WebhookSecrets{WebhookSecret: "test-secret"}},
		server.WebhookConfig{ExternalWebhookURL: "https://minder.example.com/webhook"},
		server.GitConfig{},
	)
	require.NoError(t, err)

	t.Run("supported gitlab provider class returns class info", func(t *testing.T) {
		t.Parallel()

		info, err := mgr.GetProviderClassInfo(db.ProviderClassGitlab)
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.NotEmpty(t, info.Description)
	})

	t.Run("unsupported provider class returns error", func(t *testing.T) {
		t.Parallel()

		info, err := mgr.GetProviderClassInfo(db.ProviderClassGithub)
		require.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "provider does not implement github")
	})
}

func TestMarshallConfig(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockStore := mockdb.NewMockStore(ctrl)
	mockCrypt := mockcrypto.NewMockEngine(ctrl)
	mockPub := mockinterfaces.NewMockPublisher(ctrl)

	mgr, err := NewGitLabProviderClassManager(
		context.Background(),
		mockCrypt,
		mockStore,
		mockPub,
		&server.GitLabConfig{WebhookSecrets: server.WebhookSecrets{WebhookSecret: "test-secret"}},
		server.WebhookConfig{ExternalWebhookURL: "https://minder.example.com/webhook"},
		server.GitConfig{},
	)
	require.NoError(t, err)

	t.Run("unsupported class returns error", func(t *testing.T) {
		t.Parallel()

		raw, err := mgr.MarshallConfig(context.Background(), db.ProviderClassGithub, json.RawMessage(`{}`))
		require.Error(t, err)
		assert.Nil(t, raw)
		assert.Contains(t, err.Error(), "provider does not implement github")
	})

	t.Run("supported gitlab class marshals valid empty or custom config", func(t *testing.T) {
		t.Parallel()

		raw, err := mgr.MarshallConfig(context.Background(), db.ProviderClassGitlab, json.RawMessage(`{}`))
		require.NoError(t, err)
		assert.NotNil(t, raw)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	mgr := &providerClassManager{}
	err := mgr.Delete(context.Background(), &db.Provider{})
	assert.NoError(t, err)
}

func TestBuild(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockStore := mockdb.NewMockStore(ctrl)
	mockCrypt := mockcrypto.NewMockEngine(ctrl)
	mockPub := mockinterfaces.NewMockPublisher(ctrl)

	mgr, err := NewGitLabProviderClassManager(
		context.Background(),
		mockCrypt,
		mockStore,
		mockPub,
		&server.GitLabConfig{WebhookSecrets: server.WebhookSecrets{WebhookSecret: "test-secret"}},
		server.WebhookConfig{ExternalWebhookURL: "https://minder.example.com/webhook"},
		server.GitConfig{},
	)
	require.NoError(t, err)

	t.Run("invalid provider class returns error", func(t *testing.T) {
		t.Parallel()

		prov := &db.Provider{
			Class:   db.ProviderClassGithub,
			Version: v1.V1,
		}
		res, err := mgr.Build(context.Background(), prov)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "provider does not implement gitlab")
	})

	t.Run("unsupported provider version returns error", func(t *testing.T) {
		t.Parallel()

		prov := &db.Provider{
			Class:   db.ProviderClassGitlab,
			Version: "v2",
		}
		res, err := mgr.Build(context.Background(), prov)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "provider version not supported")
	})

	t.Run("credential fetch failure returns error", func(t *testing.T) {
		t.Parallel()

		ctrlSub := gomock.NewController(t)
		storeSub := mockdb.NewMockStore(ctrlSub)

		subMgr := &providerClassManager{
			store:                  storeSub,
			crypteng:               mockCrypt,
			pub:                    mockPub,
			webhookURL:             "https://example.com",
			currentWebhookSecret:   "secret",
			previousWebhookSecrets: nil,
		}

		projID := uuid.New()
		prov := &db.Provider{
			Name:      "test-gitlab-prov",
			Class:     db.ProviderClassGitlab,
			Version:   v1.V1,
			ProjectID: projID,
		}

		storeSub.EXPECT().GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
			Provider:  "test-gitlab-prov",
			ProjectID: projID,
		}).Return(db.ProviderAccessToken{}, sql.ErrNoRows)

		res, err := subMgr.Build(context.Background(), prov)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "unable to fetch credentials")
	})

	t.Run("invalid encrypted access token returns error", func(t *testing.T) {
		t.Parallel()

		ctrlSub := gomock.NewController(t)
		storeSub := mockdb.NewMockStore(ctrlSub)

		subMgr := &providerClassManager{
			store:                  storeSub,
			crypteng:               mockCrypt,
			pub:                    mockPub,
			webhookURL:             "https://example.com",
			currentWebhookSecret:   "secret",
			previousWebhookSecrets: nil,
		}

		projID := uuid.New()
		prov := &db.Provider{
			Name:      "test-gitlab-invalid-token",
			Class:     db.ProviderClassGitlab,
			Version:   v1.V1,
			ProjectID: projID,
		}

		// EncryptedAccessToken Valid is false
		storeSub.EXPECT().GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
			Provider:  "test-gitlab-invalid-token",
			ProjectID: projID,
		}).Return(db.ProviderAccessToken{
			Provider:             "test-gitlab-invalid-token",
			EncryptedAccessToken: pqtype.NullRawMessage{Valid: false},
		}, nil)

		res, err := subMgr.Build(context.Background(), prov)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "no secret found for provider test-gitlab-invalid-token")
	})

	t.Run("malformed encrypted token data returns error", func(t *testing.T) {
		t.Parallel()

		ctrlSub := gomock.NewController(t)
		storeSub := mockdb.NewMockStore(ctrlSub)

		subMgr := &providerClassManager{
			store:                  storeSub,
			crypteng:               mockCrypt,
			pub:                    mockPub,
			webhookURL:             "https://example.com",
			currentWebhookSecret:   "secret",
			previousWebhookSecrets: nil,
		}

		projID := uuid.New()
		prov := &db.Provider{
			Name:      "test-gitlab-malformed",
			Class:     db.ProviderClassGitlab,
			Version:   v1.V1,
			ProjectID: projID,
		}

		storeSub.EXPECT().GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
			Provider:  "test-gitlab-malformed",
			ProjectID: projID,
		}).Return(db.ProviderAccessToken{
			Provider: "test-gitlab-malformed",
			EncryptedAccessToken: pqtype.NullRawMessage{
				RawMessage: []byte(`not-valid-encrypted-data`),
				Valid:      true,
			},
		}, nil)

		res, err := subMgr.Build(context.Background(), prov)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "unable to fetch credentials")
	})

	t.Run("decryption failure returns error", func(t *testing.T) {
		t.Parallel()

		ctrlSub := gomock.NewController(t)
		storeSub := mockdb.NewMockStore(ctrlSub)
		cryptSub := mockcrypto.NewMockEngine(ctrlSub)

		subMgr := &providerClassManager{
			store:                  storeSub,
			crypteng:               cryptSub,
			pub:                    mockPub,
			webhookURL:             "https://example.com",
			currentWebhookSecret:   "secret",
			previousWebhookSecrets: nil,
		}

		projID := uuid.New()
		prov := &db.Provider{
			Name:      "test-gitlab-decrypt-fail",
			Class:     db.ProviderClassGitlab,
			Version:   v1.V1,
			ProjectID: projID,
		}

		encData := crypto.EncryptedData{
			EncodedData: "some-data",
			Algorithm:   "aes-256-gcm",
		}
		rawMsg, err := encData.Serialize()
		require.NoError(t, err)

		storeSub.EXPECT().GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
			Provider:  "test-gitlab-decrypt-fail",
			ProjectID: projID,
		}).Return(db.ProviderAccessToken{
			Provider: "test-gitlab-decrypt-fail",
			EncryptedAccessToken: pqtype.NullRawMessage{
				RawMessage: rawMsg,
				Valid:      true,
			},
		}, nil)

		cryptSub.EXPECT().DecryptOAuthToken(gomock.Any()).
			Return(oauth2.Token{}, errors.New("key decryption failed"))

		res, err := subMgr.Build(context.Background(), prov)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "error decrypting access token")
	})
}
