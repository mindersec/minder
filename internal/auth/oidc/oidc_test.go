// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/oidc"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

func TestNewOIDCProvider(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 "https://public.auth.example.com",
			"authorization_endpoint": "https://public.auth.example.com/auth",
			"token_endpoint":         "https://public.auth.example.com/token",
			"jwks_uri":               "https://public.auth.example.com/jwks",
		})
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	ctx := context.Background()

	tests := []struct {
		name            string
		idpName         string
		cfg             serverconfig.IdentityConfig
		wantName        string
		wantIssuer      string
		wantErrContains string
	}{
		{
			name:    "successful discovery with matching public issuer_claim",
			idpName: "custom-oidc",
			cfg: serverconfig.IdentityConfig{
				IssuerUrl:   ts.URL,
				IssuerClaim: "https://public.auth.example.com",
			},
			wantName:   "custom-oidc",
			wantIssuer: "https://public.auth.example.com",
		},
		{
			name:    "successful discovery defaults to oidc provider name",
			idpName: "",
			cfg: serverconfig.IdentityConfig{
				IssuerUrl:   ts.URL,
				IssuerClaim: "https://public.auth.example.com",
			},
			wantName:   "oidc",
			wantIssuer: "https://public.auth.example.com",
		},
		{
			name:            "missing issuer url returns error",
			cfg:             serverconfig.IdentityConfig{},
			wantErrContains: "issuer_url is required",
		},
		{
			name: "discovery endpoint unreachable",
			cfg: serverconfig.IdentityConfig{
				IssuerUrl: "http://127.0.0.1:0",
			},
			wantErrContains: "failed to discover OIDC configuration",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider, err := oidc.NewOIDCProvider(ctx, tc.idpName, tc.cfg, ts.Client())
			if tc.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantName, provider.String())
			u := provider.URL()
			assert.Equal(t, tc.wantIssuer, u.String())
		})
	}
}

func TestProvider_Validate(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": "http://" + r.Host,
		})
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	ctx := context.Background()
	provider, err := oidc.NewOIDCProvider(ctx, "test-oidc", serverconfig.IdentityConfig{
		IssuerUrl: ts.URL,
	}, ts.Client())
	require.NoError(t, err)

	now := time.Now()

	buildToken := func(t *testing.T, sub string, claims map[string]any) jwt.Token {
		t.Helper()
		b := jwt.NewBuilder().IssuedAt(now)
		if sub != "" {
			b.Subject(sub)
		}
		for k, v := range claims {
			b.Claim(k, v)
		}
		tok, err := b.Build()
		require.NoError(t, err)
		return tok
	}

	tests := []struct {
		name          string
		token         jwt.Token
		wantUserID    string
		wantHumanName string
		wantErrIs     error
	}{
		{
			name: "preferred_username claim extraction",
			token: buildToken(t, "sub-1234", map[string]any{
				"preferred_username": "testuser",
			}),
			wantUserID:    "sub-1234",
			wantHumanName: "testuser",
		},
		{
			name: "fallback to email claim",
			token: buildToken(t, "sub-5678", map[string]any{
				"email": "user@example.com",
			}),
			wantUserID:    "sub-5678",
			wantHumanName: "user@example.com",
		},
		{
			name:          "fallback to subject when username claims absent",
			token:         buildToken(t, "sub-only", nil),
			wantUserID:    "sub-only",
			wantHumanName: "sub-only",
		},
		{
			name: "missing subject returns error",
			token: buildToken(t, "", map[string]any{
				"preferred_username": "no-sub-user",
			}),
			wantErrIs: oidc.ErrMissingSubject,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identity, err := provider.Validate(ctx, tc.token)
			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantUserID, identity.UserID)
			assert.Equal(t, tc.wantHumanName, identity.HumanName)
			assert.Equal(t, provider, identity.Provider)
		})
	}
}

func TestProvider_Resolvers(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": "http://" + r.Host,
		})
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	ctx := context.Background()
	provider, err := oidc.NewOIDCProvider(ctx, "test-oidc", serverconfig.IdentityConfig{
		IssuerUrl: ts.URL,
	}, ts.Client())
	require.NoError(t, err)

	t.Run("resolve returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		_, err := provider.Resolve(ctx, "sub-123")
		require.ErrorIs(t, err, auth.ErrNotFound)
	})

	t.Run("resolve federated returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		_, err := provider.ResolveFederated(ctx, "github", "12345")
		require.ErrorIs(t, err, auth.ErrNotFound)
	})
}
