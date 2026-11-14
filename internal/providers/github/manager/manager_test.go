// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/crypto/algorithms"
	mockcrypto "github.com/mindersec/minder/internal/crypto/mock"
	"github.com/mindersec/minder/internal/db"
	entmodels "github.com/mindersec/minder/internal/entities/models"
	mockpropssvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/github/clients"
	mockgh "github.com/mindersec/minder/internal/providers/github/mock"
	mockghsvc "github.com/mindersec/minder/internal/providers/github/service/mock"
	m "github.com/mindersec/minder/internal/providers/manager"
	"github.com/mindersec/minder/internal/providers/ratecache"
	"github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/pkg/config/server"
	mockinterfaces "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestNewGitHubProviderClassManager(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockCrypto := mockcrypto.NewMockEngine(ctrl)
	mockStore := mockdb.NewMockStore(ctrl)
	mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)
	mockPropsSvc := mockpropssvc.NewMockPropertiesService(ctrl)
	mockPublisher := mockinterfaces.NewMockPublisher(ctrl)

	mgr := NewGitHubProviderClassManager(
		ratecache.NewRestClientCache(t.Context()),
		nil,
		&server.ProviderConfig{},
		&server.WebhookConfig{},
		nil,
		mockCrypto,
		mockStore,
		mockGhSvc,
		mockPropsSvc,
		metrics.NewNoopMetrics(),
		mockPublisher,
	)

	require.NotNil(t, mgr)
	assert.ElementsMatch(t, []db.ProviderClass{db.ProviderClassGithubApp, db.ProviderClassGithub}, mgr.GetSupportedClasses())
}

func TestGetSupportedClasses(t *testing.T) {
	t.Parallel()

	mgr := &githubProviderManager{}
	classes := mgr.GetSupportedClasses()

	assert.Len(t, classes, 2)
	assert.Contains(t, classes, db.ProviderClassGithubApp)
	assert.Contains(t, classes, db.ProviderClassGithub)
}

func TestGetProviderClassInfo(t *testing.T) {
	t.Parallel()

	mgr := &githubProviderManager{}

	testCases := []struct {
		name          string
		class         db.ProviderClass
		expectedError string
	}{
		{
			name:  "github provider class",
			class: db.ProviderClassGithub,
		},
		{
			name:  "github app provider class",
			class: db.ProviderClassGithubApp,
		},
		{
			name:          "unsupported provider class dockerhub",
			class:         db.ProviderClassDockerhub,
			expectedError: "provider does not implement dockerhub",
		},
		{
			name:          "unsupported provider class quay",
			class:         db.ProviderClassQuay,
			expectedError: "provider does not implement quay",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			info, err := mgr.GetProviderClassInfo(tc.class)
			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, info)
			} else {
				require.NoError(t, err)
				require.NotNil(t, info)
			}
		})
	}
}

func TestNewOAuthConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		providerClass db.ProviderClass
		cli           bool
		setupConfig   func() *server.ProviderConfig
		expectedError string
		checkConfig   func(t *testing.T, cfg *oauth2.Config)
	}{
		{
			name:          "github oauth config for web flow",
			providerClass: db.ProviderClassGithub,
			cli:           false,
			setupConfig: func() *server.ProviderConfig {
				return &server.ProviderConfig{
					GitHub: &server.GitHubConfig{
						OAuthClientConfig: server.OAuthClientConfig{
							ClientID:     "gh-client-id",
							ClientSecret: "gh-client-secret",
							RedirectURI:  "https://example.com/oauth/callback",
						},
					},
				}
			},
			checkConfig: func(t *testing.T, cfg *oauth2.Config) {
				t.Helper()
				assert.Equal(t, "gh-client-id", cfg.ClientID)
				assert.Equal(t, "gh-client-secret", cfg.ClientSecret)
				assert.Equal(t, "https://example.com/oauth/callback/web", cfg.RedirectURL)
				assert.ElementsMatch(t, []string{"user:email", "repo", "read:packages", "write:packages", "workflow", "read:org"}, cfg.Scopes)
			},
		},
		{
			name:          "github oauth config for cli flow",
			providerClass: db.ProviderClassGithub,
			cli:           true,
			setupConfig: func() *server.ProviderConfig {
				return &server.ProviderConfig{
					GitHub: &server.GitHubConfig{
						OAuthClientConfig: server.OAuthClientConfig{
							ClientID:     "gh-client-id-cli",
							ClientSecret: "gh-client-secret-cli",
							RedirectURI:  "https://example.com/oauth/callback",
						},
					},
				}
			},
			checkConfig: func(t *testing.T, cfg *oauth2.Config) {
				t.Helper()
				assert.Equal(t, "gh-client-id-cli", cfg.ClientID)
				assert.Equal(t, "gh-client-secret-cli", cfg.ClientSecret)
				assert.Equal(t, "https://example.com/oauth/callback/cli", cfg.RedirectURL)
				assert.ElementsMatch(t, []string{"user:email", "repo", "read:packages", "write:packages", "workflow", "read:org"}, cfg.Scopes)
			},
		},
		{
			name:          "github app oauth config",
			providerClass: db.ProviderClassGithubApp,
			cli:           false,
			setupConfig: func() *server.ProviderConfig {
				return &server.ProviderConfig{
					GitHubApp: &server.GitHubAppConfig{
						OAuthClientConfig: server.OAuthClientConfig{
							ClientID:     "gh-app-client-id",
							ClientSecret: "gh-app-client-secret",
							RedirectURI:  "https://example.com/github-app/callback",
						},
					},
				}
			},
			checkConfig: func(t *testing.T, cfg *oauth2.Config) {
				t.Helper()
				assert.Equal(t, "gh-app-client-id", cfg.ClientID)
				assert.Equal(t, "gh-app-client-secret", cfg.ClientSecret)
				assert.Equal(t, "https://example.com/github-app/callback", cfg.RedirectURL)
				assert.Empty(t, cfg.Scopes)
			},
		},
		{
			name:          "custom endpoint token URL override",
			providerClass: db.ProviderClassGithub,
			cli:           false,
			setupConfig: func() *server.ProviderConfig {
				return &server.ProviderConfig{
					GitHub: &server.GitHubConfig{
						OAuthClientConfig: server.OAuthClientConfig{
							ClientID:     "gh-client-id",
							ClientSecret: "gh-client-secret",
							RedirectURI:  "https://example.com/oauth/callback",
							Endpoint: &server.OAuthEndpoint{
								TokenURL: "https://custom-auth.example.com/token",
							},
						},
					},
				}
			},
			checkConfig: func(t *testing.T, cfg *oauth2.Config) {
				t.Helper()
				assert.Equal(t, "https://custom-auth.example.com/token", cfg.Endpoint.TokenURL)
			},
		},
		{
			name:          "unsupported provider class",
			providerClass: db.ProviderClassDockerhub,
			cli:           false,
			setupConfig: func() *server.ProviderConfig {
				return &server.ProviderConfig{}
			},
			expectedError: "invalid provider class: dockerhub",
		},
		{
			name:          "missing client id file error",
			providerClass: db.ProviderClassGithub,
			cli:           false,
			setupConfig: func() *server.ProviderConfig {
				return &server.ProviderConfig{
					GitHub: &server.GitHubConfig{
						OAuthClientConfig: server.OAuthClientConfig{
							ClientIDFile: "non-existent-client-id-file.txt",
							ClientSecret: "gh-secret",
							RedirectURI:  "https://example.com/callback",
						},
					},
				}
			},
			expectedError: "failed to get client ID",
		},
		{
			name:          "missing client secret file error",
			providerClass: db.ProviderClassGithub,
			cli:           false,
			setupConfig: func() *server.ProviderConfig {
				return &server.ProviderConfig{
					GitHub: &server.GitHubConfig{
						OAuthClientConfig: server.OAuthClientConfig{
							ClientID:         "gh-id",
							ClientSecretFile: "non-existent-client-secret-file.txt",
							RedirectURI:      "https://example.com/callback",
						},
					},
				}
			},
			expectedError: "failed to get client secret",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mgr := &githubProviderManager{
				config: tc.setupConfig(),
			}

			cfg, err := mgr.NewOAuthConfig(tc.providerClass, tc.cli)
			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				if tc.checkConfig != nil {
					tc.checkConfig(t, cfg)
				}
			}
		})
	}
}

func TestMarshallConfig(t *testing.T) {
	t.Parallel()

	mgr := &githubProviderManager{}

	testCases := []struct {
		name          string
		class         db.ProviderClass
		inputJSON     string
		expectedError string
	}{
		{
			name:      "valid github oauth config",
			class:     db.ProviderClassGithub,
			inputJSON: `{}`,
		},
		{
			name:      "valid github app config",
			class:     db.ProviderClassGithubApp,
			inputJSON: `{}`,
		},
		{
			name:          "unsupported provider class",
			class:         db.ProviderClassDockerhub,
			inputJSON:     `{}`,
			expectedError: "provider does not implement dockerhub",
		},
		{
			name:          "malformed JSON github class",
			class:         db.ProviderClassGithub,
			inputJSON:     `{invalid-json`,
			expectedError: "invalid character",
		},
		{
			name:          "malformed JSON github app class",
			class:         db.ProviderClassGithubApp,
			inputJSON:     `{invalid-json`,
			expectedError: "invalid character",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := mgr.MarshallConfig(context.Background(), tc.class, json.RawMessage(tc.inputJSON))
			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}
}

type dummyCredential struct{}

func (dummyCredential) SetToken(_ string) {}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()

	t.Run("invalid credential type returns error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)
		mgr := &githubProviderManager{ghService: mockGhSvc}

		err := mgr.ValidateCredentials(t.Context(), dummyCredential{}, &m.CredentialVerifyParams{RemoteUser: "test-user"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credential type")
	})

	t.Run("empty remote user skips token verification", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)
		mgr := &githubProviderManager{ghService: mockGhSvc}

		tokenCred := credentials.NewOAuth2TokenCredential("valid-token")
		err := mgr.ValidateCredentials(t.Context(), tokenCred, &m.CredentialVerifyParams{RemoteUser: ""})
		require.NoError(t, err)
	})

	t.Run("valid remote user with successful token verification", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)
		mgr := &githubProviderManager{ghService: mockGhSvc}

		mockGhSvc.EXPECT().
			VerifyProviderTokenIdentity(gomock.Any(), "octocat", "valid-token").
			Return(nil)

		tokenCred := credentials.NewOAuth2TokenCredential("valid-token")
		err := mgr.ValidateCredentials(t.Context(), tokenCred, &m.CredentialVerifyParams{RemoteUser: "octocat"})
		require.NoError(t, err)
	})

	t.Run("token identity verification failure returns error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)
		mgr := &githubProviderManager{ghService: mockGhSvc}

		mockGhSvc.EXPECT().
			VerifyProviderTokenIdentity(gomock.Any(), "octocat", "bad-token").
			Return(errors.New("identity mismatch"))

		tokenCred := credentials.NewOAuth2TokenCredential("bad-token")
		err := mgr.ValidateCredentials(t.Context(), tokenCred, &m.CredentialVerifyParams{RemoteUser: "octocat"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error verifying token identity: identity mismatch")
	})
}

func TestBuild_ValidationErrors(t *testing.T) {
	t.Parallel()

	mgr := &githubProviderManager{}
	ctx := context.Background()

	t.Run("unsupported provider class", func(t *testing.T) {
		t.Parallel()

		provConfig := &db.Provider{
			Class:   db.ProviderClassDockerhub,
			Version: v1.V1,
		}

		p, err := mgr.Build(ctx, provConfig)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider does not implement github")
		assert.Nil(t, p)
	})

	t.Run("unsupported provider version", func(t *testing.T) {
		t.Parallel()

		provConfig := &db.Provider{
			Class:   db.ProviderClassGithub,
			Version: "v2",
		}

		p, err := mgr.Build(ctx, provConfig)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider version not supported")
		assert.Nil(t, p)
	})
}

func TestGetWebhookHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockPublisher := mockinterfaces.NewMockPublisher(ctrl)
	mgr := &githubProviderManager{
		mt:        metrics.NewNoopMetrics(),
		publisher: mockPublisher,
		whconfig: &server.WebhookConfig{
			ExternalWebhookURL: "https://example.com/webhook",
		},
	}

	handler := mgr.GetWebhookHandler()
	require.NotNil(t, handler)
}

func helperGenerateEncryptedToken(t *testing.T) pqtype.NullRawMessage {
	t.Helper()

	data := crypto.EncryptedData{
		Algorithm:   algorithms.Aes256Gcm,
		EncodedData: "dnS6VFiMYrfnbeP6eixmBw==",
		KeyVersion:  "",
	}

	serialized, err := data.Serialize()
	require.NoError(t, err)

	return pqtype.NullRawMessage{
		RawMessage: serialized,
		Valid:      true,
	}
}

func helperCreatePrivateKeyFile(t *testing.T) *os.File {
	t.Helper()

	tempDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tempDir, "gh-app-pvtkey-*.pem")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tmpFile.Close()
	})

	pvtKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	err = pem.Encode(tmpFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(pvtKey),
	})
	require.NoError(t, err)

	return tmpFile
}

func TestBuild_OAuthAndAppFlows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	providerID := uuid.New()
	projectID := uuid.New()
	accessToken := "gh-access-token"

	t.Run("github oauth provider build successful from scratch", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		mockCrypto := mockcrypto.NewMockEngine(ctrl)

		prov := &db.Provider{
			ID:         providerID,
			ProjectID:  projectID,
			Name:       "gh-provider-oauth",
			Class:      db.ProviderClassGithub,
			Version:    v1.V1,
			Definition: json.RawMessage(`{}`),
		}

		encToken := db.ProviderAccessToken{
			EncryptedAccessToken: helperGenerateEncryptedToken(t),
			OwnerFilter:          sql.NullString{String: "my-org", Valid: true},
		}

		mockStore.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
				Provider:  prov.Name,
				ProjectID: prov.ProjectID,
			}).
			Return(encToken, nil).Times(2)

		mockCrypto.EXPECT().
			DecryptOAuthToken(gomock.Any()).
			Return(oauth2.Token{AccessToken: accessToken}, nil).Times(2)

		clientCache := ratecache.NewRestClientCache(t.Context())
		ghFactory := clients.NewGitHubClientFactory(telemetry.NewNoopMetrics())

		mgr := &githubProviderManager{
			store:           mockStore,
			crypteng:        mockCrypto,
			restClientCache: clientCache,
			ghClientFactory: ghFactory,
			config:          &server.ProviderConfig{},
		}

		client, err := mgr.Build(ctx, prov)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Subsequent call should hit restClientCache.Get
		cachedClient, err := mgr.Build(ctx, prov)
		require.NoError(t, err)
		assert.Equal(t, client, cachedClient)
	})

	t.Run("github oauth provider token not found error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)

		prov := &db.Provider{
			ID:        providerID,
			ProjectID: projectID,
			Name:      "gh-provider-missing-token",
			Class:     db.ProviderClassGithub,
			Version:   v1.V1,
		}

		mockStore.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
			Return(db.ProviderAccessToken{}, sql.ErrNoRows)

		mgr := &githubProviderManager{
			store: mockStore,
		}

		client, err := mgr.Build(ctx, prov)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no access token")
		assert.Nil(t, client)
	})

	t.Run("github oauth provider backwards compatible encrypted token fallback", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		mockCrypto := mockcrypto.NewMockEngine(ctrl)

		prov := &db.Provider{
			ID:         providerID,
			ProjectID:  projectID,
			Name:       "gh-provider-legacy",
			Class:      db.ProviderClassGithub,
			Version:    v1.V1,
			Definition: json.RawMessage(`{}`),
		}

		encToken := db.ProviderAccessToken{
			EncryptedToken: sql.NullString{String: "legacy-encrypted-string", Valid: true},
		}

		mockStore.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
			Return(encToken, nil)

		mockCrypto.EXPECT().
			DecryptOAuthToken(gomock.Any()).
			Return(oauth2.Token{AccessToken: accessToken}, nil)

		clientCache := ratecache.NewRestClientCache(t.Context())
		ghFactory := clients.NewGitHubClientFactory(telemetry.NewNoopMetrics())

		mgr := &githubProviderManager{
			store:           mockStore,
			crypteng:        mockCrypto,
			restClientCache: clientCache,
			ghClientFactory: ghFactory,
			config:          &server.ProviderConfig{},
		}

		client, err := mgr.Build(ctx, prov)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("github oauth provider token decryption error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		mockCrypto := mockcrypto.NewMockEngine(ctrl)

		prov := &db.Provider{
			ID:        providerID,
			ProjectID: projectID,
			Name:      "gh-provider-decrypt-fail",
			Class:     db.ProviderClassGithub,
			Version:   v1.V1,
		}

		encToken := db.ProviderAccessToken{
			EncryptedAccessToken: helperGenerateEncryptedToken(t),
		}

		mockStore.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
			Return(encToken, nil)

		mockCrypto.EXPECT().
			DecryptOAuthToken(gomock.Any()).
			Return(oauth2.Token{}, errors.New("decryption failed"))

		mgr := &githubProviderManager{
			store:    mockStore,
			crypteng: mockCrypto,
		}

		client, err := mgr.Build(ctx, prov)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error decrypting access token: decryption failed")
		assert.Nil(t, client)
	})

	t.Run("github oauth provider no secret found error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)

		prov := &db.Provider{
			ID:        providerID,
			ProjectID: projectID,
			Name:      "gh-provider-no-secret",
			Class:     db.ProviderClassGithub,
			Version:   v1.V1,
		}

		encToken := db.ProviderAccessToken{
			Provider: "gh-provider-no-secret",
		}

		mockStore.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), gomock.Any()).
			Return(encToken, nil)

		mgr := &githubProviderManager{
			store: mockStore,
		}

		client, err := mgr.Build(ctx, prov)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no secret found for provider gh-provider-no-secret")
		assert.Nil(t, client)
	})

	t.Run("github app provider build successful", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		keyFile := helperCreatePrivateKeyFile(t)

		prov := &db.Provider{
			ID:         providerID,
			ProjectID:  projectID,
			Name:       "gh-app-provider",
			Class:      db.ProviderClassGithubApp,
			Version:    v1.V1,
			Definition: json.RawMessage(`{}`),
		}

		mockStore.EXPECT().
			GetInstallationIDByProviderID(gomock.Any(), uuid.NullUUID{UUID: providerID, Valid: true}).
			Return(db.ProviderGithubAppInstallation{
				AppInstallationID: 12345,
				IsOrg:             true,
			}, nil)

		clientCache := ratecache.NewRestClientCache(t.Context())
		ghFactory := clients.NewGitHubClientFactory(telemetry.NewNoopMetrics())

		mgr := &githubProviderManager{
			store:           mockStore,
			restClientCache: clientCache,
			ghClientFactory: ghFactory,
			config: &server.ProviderConfig{
				GitHubApp: &server.GitHubAppConfig{
					AppID:      67890,
					AppName:    "minder-app",
					PrivateKey: keyFile.Name(),
				},
			},
		}

		client, err := mgr.Build(ctx, prov)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("github app provider no known installation sets empty credential", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)

		prov := &db.Provider{
			ID:         providerID,
			ProjectID:  projectID,
			Name:       "gh-app-no-install",
			Class:      db.ProviderClassGithubApp,
			Version:    v1.V1,
			Definition: json.RawMessage(`{}`),
		}

		mockStore.EXPECT().
			GetInstallationIDByProviderID(gomock.Any(), uuid.NullUUID{UUID: providerID, Valid: true}).
			Return(db.ProviderGithubAppInstallation{}, sql.ErrNoRows)

		clientCache := ratecache.NewRestClientCache(t.Context())
		ghFactory := clients.NewGitHubClientFactory(telemetry.NewNoopMetrics())

		mgr := &githubProviderManager{
			store:           mockStore,
			restClientCache: clientCache,
			ghClientFactory: ghFactory,
			config: &server.ProviderConfig{
				GitHubApp: &server.GitHubAppConfig{
					AppName: "minder-app",
				},
			},
		}

		client, err := mgr.Build(ctx, prov)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("github app provider installation lookup error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)

		prov := &db.Provider{
			ID:        providerID,
			ProjectID: projectID,
			Name:      "gh-app-error",
			Class:     db.ProviderClassGithubApp,
			Version:   v1.V1,
		}

		mockStore.EXPECT().
			GetInstallationIDByProviderID(gomock.Any(), uuid.NullUUID{UUID: providerID, Valid: true}).
			Return(db.ProviderGithubAppInstallation{}, errors.New("db error"))

		mgr := &githubProviderManager{
			store: mockStore,
		}

		client, err := mgr.Build(ctx, prov)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting installation ID: db error")
		assert.Nil(t, client)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	providerID := uuid.New()
	projectID := uuid.New()

	t.Run("credential state not set only deletes installation", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)

		// Provider with auth flow none has CredentialStateNotApplicable (not CredentialStateSet)
		prov := &db.Provider{
			ID:        providerID,
			ProjectID: projectID,
			AuthFlows: []db.AuthorizationFlow{db.AuthorizationFlowNone},
		}

		mockGhSvc.EXPECT().
			DeleteInstallation(gomock.Any(), providerID).
			Return(nil)

		mgr := &githubProviderManager{
			store:     mockStore,
			ghService: mockGhSvc,
		}

		err := mgr.Delete(ctx, prov)
		require.NoError(t, err)
	})

	t.Run("credential state error returns error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)

		prov := &db.Provider{
			ID:        providerID,
			ProjectID: projectID,
			Class:     db.ProviderClassGithub,
		}

		mockStore.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
				Provider:  prov.Name,
				ProjectID: prov.ProjectID,
			}).
			Return(db.ProviderAccessToken{}, errors.New("db failure"))

		mgr := &githubProviderManager{
			store:     mockStore,
			ghService: mockGhSvc,
		}

		err := mgr.Delete(ctx, prov)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to get credential state for provider")
	})

	t.Run("credential state set successfully deregisters entities and deletes installation", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)
		mockCrypto := mockcrypto.NewMockEngine(ctrl)
		mockPropsSvc := mockpropssvc.NewMockPropertiesService(ctrl)
		mockGhClient := mockgh.NewMockGitHub(ctrl)

		accessToken := "test-access-token"
		prov := &db.Provider{
			ID:         providerID,
			ProjectID:  projectID,
			Name:       "github-provider",
			Class:      db.ProviderClassGithub,
			Version:    v1.V1,
			Definition: json.RawMessage(`{}`),
		}

		encToken := db.ProviderAccessToken{
			EncryptedAccessToken: helperGenerateEncryptedToken(t),
		}

		// Mock calls for credential checking
		mockStore.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
				Provider:  prov.Name,
				ProjectID: prov.ProjectID,
			}).
			Return(encToken, nil).Times(2) // Once in GetCredentialStateForProvider, once in Build

		mockCrypto.EXPECT().
			DecryptOAuthToken(gomock.Any()).
			Return(oauth2.Token{AccessToken: accessToken}, nil).Times(2)

		// Setup rest client cache with mock github client
		clientCache := ratecache.NewRestClientCache(t.Context())
		clientCache.Set("", accessToken, db.ProviderTypeGithub, mockGhClient)

		// Mock store returning entities to deregister
		entityID := uuid.New()
		mockStore.EXPECT().
			GetEntitiesByProvider(gomock.Any(), providerID).
			Return([]db.EntityInstance{
				{
					ID:         entityID,
					Name:       "repo-1",
					ProviderID: providerID,
					ProjectID:  projectID,
				},
			}, nil)

		// Mock props service returning valid EntityWithProperties
		mockPropsSvc.EXPECT().
			EntityWithPropertiesByID(gomock.Any(), entityID, nil).
			Return(entmodels.NewEntityWithPropertiesFromInstance(entmodels.EntityInstance{}, nil), nil)

		// Mock provider DeregisterEntity call
		mockGhClient.EXPECT().
			DeregisterEntity(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		// Mock DeleteInstallation call
		mockGhSvc.EXPECT().
			DeleteInstallation(gomock.Any(), providerID).
			Return(nil)

		mgr := &githubProviderManager{
			store:           mockStore,
			crypteng:        mockCrypto,
			propsSvc:        mockPropsSvc,
			ghService:       mockGhSvc,
			restClientCache: clientCache,
			config:          &server.ProviderConfig{},
		}

		err := mgr.Delete(ctx, prov)
		require.NoError(t, err)
	})

	t.Run("installation deletion error returns error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockStore := mockdb.NewMockStore(ctrl)
		mockGhSvc := mockghsvc.NewMockGitHubProviderService(ctrl)

		prov := &db.Provider{
			ID:        providerID,
			ProjectID: projectID,
			AuthFlows: []db.AuthorizationFlow{db.AuthorizationFlowNone},
		}

		mockGhSvc.EXPECT().
			DeleteInstallation(gomock.Any(), providerID).
			Return(errors.New("failed to delete installation"))

		mgr := &githubProviderManager{
			store:     mockStore,
			ghService: mockGhSvc,
		}

		err := mgr.Delete(ctx, prov)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete installation")
	})
}
