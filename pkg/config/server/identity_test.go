// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityConfig_DiscoverOIDCEndpoints(t *testing.T) {
	t.Parallel()

	// Mock OIDC discovery server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/stacklok/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]interface{}{
				"issuer":         "http://example.com/realms/stacklok",
				"jwks_uri":       "http://example.com/realms/stacklok/keys",
				"token_endpoint": "http://example.com/realms/stacklok/token",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	cfg := &IdentityConfig{
		IssuerUrl:    testServer.URL,
		Realm:        "stacklok",
		ClientId:     "client-id",
		ClientSecret: "client-secret",
	}

	oidcCfg, err := cfg.DiscoverOIDCEndpoints(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, oidcCfg)

	assert.Equal(t, "http://example.com/realms/stacklok", oidcCfg.Issuer)
	assert.Equal(t, "http://example.com/realms/stacklok/keys", oidcCfg.JWKSURI)
	assert.Equal(t, "http://example.com/realms/stacklok/token", oidcCfg.TokenURI)
}

func TestIdentityConfig_DiscoverOIDCEndpoints_InvalidURL(t *testing.T) {
	t.Parallel()

	cfg := &IdentityConfig{
		IssuerUrl: "http://invalid-url-that-does-not-exist-12345.com",
	}

	oidcCfg, err := cfg.DiscoverOIDCEndpoints(context.Background())
	assert.Error(t, err)
	assert.Nil(t, oidcCfg)
}
