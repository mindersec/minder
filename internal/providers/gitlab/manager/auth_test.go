// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/mindersec/minder/database/mock"
	mockcrypto "github.com/mindersec/minder/internal/crypto/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestNewOAuthConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cfg          *server.GitLabConfig
		cli          bool
		wantErr      bool
		errSubstring string
		validate     func(t *testing.T, cfg *oauth2.Config)
	}{
		{
			name: "valid web oauth config",
			cfg: &server.GitLabConfig{
				Scopes: []string{"api", "read_user"},
				OAuthClientConfig: server.OAuthClientConfig{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					RedirectURI:  "https://example.com/oauth/callback",
				},
			},
			cli:     false,
			wantErr: false,
			validate: func(t *testing.T, cfg *oauth2.Config) {
				t.Helper()
				require.NotNil(t, cfg)
				assert.Equal(t, "test-client-id", cfg.ClientID)
				assert.Equal(t, "test-client-secret", cfg.ClientSecret)
				assert.Equal(t, "https://example.com/oauth/callback/web", cfg.RedirectURL)
				assert.Equal(t, []string{"api", "read_user"}, cfg.Scopes)
			},
		},
		{
			name: "valid cli oauth config with custom token endpoint",
			cfg: &server.GitLabConfig{
				Scopes: []string{"read_repository"},
				OAuthClientConfig: server.OAuthClientConfig{
					ClientID:     "cli-client-id",
					ClientSecret: "cli-client-secret",
					RedirectURI:  "https://example.com/oauth/callback",
					Endpoint: &server.OAuthEndpoint{
						TokenURL: "https://gitlab.example.com/oauth/token",
					},
				},
			},
			cli:     true,
			wantErr: false,
			validate: func(t *testing.T, cfg *oauth2.Config) {
				t.Helper()
				require.NotNil(t, cfg)
				assert.Equal(t, "cli-client-id", cfg.ClientID)
				assert.Equal(t, "cli-client-secret", cfg.ClientSecret)
				assert.Equal(t, "https://example.com/oauth/callback/cli", cfg.RedirectURL)
				assert.Equal(t, "https://gitlab.example.com/oauth/token", cfg.Endpoint.TokenURL)
			},
		},
		{
			name: "missing client id file returns error",
			cfg: &server.GitLabConfig{
				OAuthClientConfig: server.OAuthClientConfig{
					ClientIDFile: "nonexistent-client-id-file.txt",
					ClientSecret: "test-secret",
					RedirectURI:  "https://example.com/callback",
				},
			},
			cli:          false,
			wantErr:      true,
			errSubstring: "failed to get client ID",
		},
		{
			name: "missing client secret file returns error",
			cfg: &server.GitLabConfig{
				OAuthClientConfig: server.OAuthClientConfig{
					ClientID:         "test-id",
					ClientSecretFile: "nonexistent-client-secret-file.txt",
					RedirectURI:      "https://example.com/callback",
				},
			},
			cli:          false,
			wantErr:      true,
			errSubstring: "failed to get client secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := &providerClassManager{
				glpcfg: tt.cfg,
			}

			cfg, err := mgr.NewOAuthConfig(db.ProviderClassGitlab, tt.cli)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstring != "" {
					assert.Contains(t, err.Error(), tt.errSubstring)
				}
				assert.Nil(t, cfg)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

type testFailingOAuth2Cred struct{}

func (*testFailingOAuth2Cred) GetCredential() string { return "" }
func (*testFailingOAuth2Cred) GetCacheKey() string   { return "" }
func (*testFailingOAuth2Cred) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return &failingTokenSource{}
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupCred    func() provv1.Credential
		wantErr      bool
		errSubstring string
	}{
		{
			name: "valid PAT string credential with glpat- prefix",
			setupCred: func() provv1.Credential {
				return "glpat-valid-pat-secret-12345"
			},
			wantErr: false,
		},
		{
			name: "invalid PAT string without glpat- prefix returns error",
			setupCred: func() provv1.Credential {
				return "invalid-token-prefix"
			},
			wantErr:      true,
			errSubstring: "expected token to start with 'glpat-'",
		},
		{
			name: "valid OAuth2TokenCredential with token source",
			setupCred: func() provv1.Credential {
				return credentials.NewOAuth2TokenCredential("valid-oauth-token")
			},
			wantErr: false,
		},
		{
			name: "OAuth2TokenCredential returning error on token retrieval",
			setupCred: func() provv1.Credential {
				return &testFailingOAuth2Cred{}
			},
			wantErr:      true,
			errSubstring: "cannot get token from credential",
		},
		{
			name: "unsupported credential type returns error",
			setupCred: func() provv1.Credential {
				return struct{}{}
			},
			wantErr:      true,
			errSubstring: "invalid credential type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cred := tt.setupCred()

			mgr := &providerClassManager{}
			err := mgr.ValidateCredentials(context.Background(), cred, nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstring != "" {
					assert.Contains(t, err.Error(), tt.errSubstring)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

type failingTokenSource struct{}

func (*failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("token retrieval failed")
}

func helperNewTestManager(
	t *testing.T,
	pub interfaces.Publisher,
	currentSecret string,
	previousSecrets []string,
) *providerClassManager {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockCrypto := mockcrypto.NewMockEngine(ctrl)
	mockStore := mockdb.NewMockStore(ctrl)

	return &providerClassManager{
		store:                  mockStore,
		crypteng:               mockCrypto,
		pub:                    pub,
		parentContext:          context.Background(),
		currentWebhookSecret:   currentSecret,
		previousWebhookSecrets: previousSecrets,
	}
}
