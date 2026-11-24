// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const hubTestPAT = "dckr_pat_test"

// hubTestJWT returns an unsigned JWT-shaped string with the given iat and exp claims.
func hubTestJWT(t *testing.T, iat, exp time.Time) string {
	t.Helper()
	payload, err := json.Marshal(map[string]int64{"iat": iat.Unix(), "exp": exp.Unix()})
	require.NoError(t, err)
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(`{"alg":"none"}`)) + "." + enc.EncodeToString(payload) + ".sig"
}

// tokenHandler serves the token exchange endpoint and records what it received.
func tokenHandler(t *testing.T, jwt string, calls *atomic.Int32) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Empty(t, r.Header.Get("Authorization"), "the exchange itself must not send the PAT as a header")

		var body struct {
			Identifier string `json:"identifier"`
			Secret     string `json:"secret"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "acme-bot", body.Identifier)
		assert.Equal(t, hubTestPAT, body.Secret)

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"access_token":%q}`, jwt)
	}
}

func TestHubTokenSource(t *testing.T) {
	t.Parallel()

	// Docker Hub's clock runs an hour ahead of ours; the token lives 10 minutes.
	iat := time.Now().Add(time.Hour).Truncate(time.Second)
	jwt := hubTestJWT(t, iat, iat.Add(10*time.Minute))
	var calls atomic.Int32
	srv := httptest.NewServer(tokenHandler(t, jwt, &calls))
	defer srv.Close()

	src := &hubTokenSource{
		cli:      srv.Client(),
		tokenURL: srv.URL,
		username: "acme-bot",
		pat:      oauth2.StaticTokenSource(&oauth2.Token{AccessToken: hubTestPAT}),
	}

	before := time.Now()
	tok, err := src.Token()
	require.NoError(t, err)
	assert.Equal(t, jwt, tok.AccessToken)
	assert.Equal(t, "Bearer", tok.TokenType)
	// The 10 minute lifetime counts from our clock, not from Docker Hub's exp.
	assert.WithinDuration(t, before.Add(10*time.Minute), tok.Expiry, 5*time.Second)
	assert.Equal(t, int32(1), calls.Load())
}

func TestHubTokenSourceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{
			name: "rejected credentials",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
			},
			wantErr: "401",
		},
		{
			name: "missing access_token",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			},
			wantErr: "no access_token",
		},
		{
			name: "oversized response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprintf(w, `{"access_token":%q}`, strings.Repeat("a", maxTokenResponseSize))
			},
			wantErr: "error decoding",
		},
		{
			name: "malformed response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`not json`))
			},
			wantErr: "error decoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			src := &hubTokenSource{
				cli:      srv.Client(),
				tokenURL: srv.URL,
				username: "acme-bot",
				pat:      oauth2.StaticTokenSource(&oauth2.Token{AccessToken: hubTestPAT}),
			}
			_, err := src.Token()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestJWTExpiry(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	enc := base64.RawURLEncoding
	withClaims := func(claims string) string {
		return enc.EncodeToString([]byte(`{"alg":"none"}`)) + "." + enc.EncodeToString([]byte(claims)) + ".sig"
	}

	tests := []struct {
		name  string
		token string
		want  time.Time
	}{
		{"lifetime from iat and exp", hubTestJWT(t, now, now.Add(10*time.Minute)), now.Add(10 * time.Minute)},
		{"issuer clock ahead of ours", hubTestJWT(t, now.Add(time.Hour), now.Add(time.Hour+10*time.Minute)), now.Add(10 * time.Minute)},
		{"issuer clock behind ours", hubTestJWT(t, now.Add(-time.Hour), now.Add(-time.Hour+10*time.Minute)), now.Add(10 * time.Minute)},
		{"long lifetime is capped", hubTestJWT(t, now, now.Add(2*time.Hour)), now.Add(hubTokenMaxTTL)},
		{"missing iat", withClaims(`{"exp":1700000600}`), now.Add(hubTokenFallbackTTL)},
		{"exp not after iat", withClaims(`{"iat":1700000600,"exp":1700000600}`), now.Add(hubTokenFallbackTTL)},
		{"not a JWT", "not-a-jwt", now.Add(hubTokenFallbackTTL)},
		{"payload not base64", "a.!!!.c", now.Add(hubTokenFallbackTTL)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, jwtExpiry(tt.token, now).Equal(tt.want), "got %v, want %v", jwtExpiry(tt.token, now), tt.want)
		})
	}
}

// ListImages must call the Docker Hub API with the exchanged JWT, not the
// personal access token, and reuse the JWT across calls until it expires.
func TestListImagesUsesExchangedToken(t *testing.T) {
	t.Parallel()

	jwt := hubTestJWT(t, time.Now(), time.Now().Add(10*time.Minute))
	var exchanges atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/auth/token", tokenHandler(t, jwt, &exchanges))
	mux.HandleFunc("/v2/repositories/acme-org", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+jwt {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"results":[{"name":"app"}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target, err := url.Parse(srv.URL + "/v2/repositories/acme-org")
	require.NoError(t, err)

	pat := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: hubTestPAT})
	dh := &dockerHubImageLister{
		cli:       newHubClient(pat, "acme-bot", srv.URL+"/v2/auth/token"),
		namespace: "acme-org",
		target:    target,
	}

	for range 2 {
		images, err := dh.ListImages(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []string{"app"}, images)
	}
	assert.Equal(t, int32(1), exchanges.Load(), "the JWT should be reused until it expires")
}
