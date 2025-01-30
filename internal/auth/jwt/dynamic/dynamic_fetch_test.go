//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package dynamic provides the logic for reading and validating JWT tokens
// using a JWKS URL from the token's
package dynamic

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/require"
)

func TestValidator_ParseAndValidate(t *testing.T) {
	t.Parallel()

	keyGen := rand.New(rand.NewSource(12345))
	key, err := rsa.GenerateKey(keyGen, 2048)
	require.NoError(t, err)
	jwkKey, err := jwk.FromRaw(key)
	require.NoError(t, err)
	require.NoError(t, jwkKey.Set(jwk.KeyIDKey, "test"))
	require.NoError(t, jwkKey.Set(jwk.AlgorithmKey, jwa.RS256))
	require.NoError(t, jwkKey.Set(jwk.KeyUsageKey, "sig"))
	pubKey, err := jwkKey.PublicKey()
	require.NoError(t, err)

	keySet := jwk.NewSet()
	keySet.AddKey(pubKey)
	keySetJSON, err := json.Marshal(keySet)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("/certs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(keySetJSON)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// We need to add this to the mux after server start, because it includes the server.URL
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
		"issuer":"%[1]s",
		"jwks_uri":"%[1]s/certs",
		"scopes_supported":["openid","email","profile"],
		"claims_supported":["sub","email","iss","aud","iat","exp"]
		}`, server.URL)))
	})
	mux.HandleFunc("/other/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
		"issuer":"%[1]s/other",
		"jwks_uri":"%[1]s/certs",
		"scopes_supported":["openid","email","profile"],
		"claims_supported":["sub","email","iss","aud","iat","exp"]
		}`, server.URL)))
	})
	mux.HandleFunc("/elsewhere/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{
		"issuer":"%[1]s/elsewhere",
		"jwks_uri":"%[1]s/non-existent",
		"scopes_supported":["openid","email","profile"],
		"claims_supported":["sub","email","iss","aud","iat","exp"]
		}`, server.URL)))
	})

	tests := []struct {
		name     string
		getToken func(t *testing.T) (string, openid.Token)
		wantErr  string
	}{{
		name: "valid token",
		getToken: func(t *testing.T) (string, openid.Token) {
			token, err := openid.NewBuilder().
				Issuer(server.URL).
				Subject("test").
				Audience([]string{"minder"}).
				Expiration(time.Now().Add(time.Minute)).
				IssuedAt(time.Now()).
				Build()
			require.NoError(t, err)
			signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkKey))
			require.NoError(t, err)
			return string(signed), token
		},
	}, {
		name: "valid token, other issuer",
		getToken: func(t *testing.T) (string, openid.Token) {
			token, err := openid.NewBuilder().
				Issuer(server.URL + "/other").
				Subject("test").
				Audience([]string{"minder"}).
				Expiration(time.Now().Add(time.Minute)).
				IssuedAt(time.Now()).
				Build()
			require.NoError(t, err)
			signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkKey))
			require.NoError(t, err)
			return string(signed), token
		},
	}, {
		name: "invalid signature",
		getToken: func(_ *testing.T) (string, openid.Token) {
			return "invalid", nil
		},
		wantErr: `failed to split compact JWT: invalid number of segments`,
	}, {
		name: "expired jwt",
		getToken: func(_ *testing.T) (string, openid.Token) {
			token, err := openid.NewBuilder().
				Issuer(server.URL + "/elsewhere").
				Subject("test").
				Audience([]string{"minder"}).
				Expiration(time.Now().Add(-1 * time.Minute)).
				IssuedAt(time.Now().Add(-2 * time.Minute)).
				Build()
			require.NoError(t, err)
			signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkKey))
			require.NoError(t, err)
			return string(signed), token
		},
		wantErr: `failed to parse JWT payload: "exp" not satisfied`,
	}, {
		name: "bad well-known URL",
		getToken: func(t *testing.T) (string, openid.Token) {
			token, err := openid.NewBuilder().
				Issuer(server.URL + "/elsewhere").
				Subject("test").
				Audience([]string{"minder"}).
				Expiration(time.Now().Add(time.Minute)).
				IssuedAt(time.Now()).
				Build()
			require.NoError(t, err)
			signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkKey))
			require.NoError(t, err)
			return string(signed), token
		},
		wantErr: `non-200 response code "404 Not Found"`,
	}, {
		name: "bad issuer",
		getToken: func(t *testing.T) (string, openid.Token) {
			token, err := openid.NewBuilder().
				Issuer(server.URL + "/nothing").
				Subject("test").
				Audience([]string{"minder"}).
				Expiration(time.Now().Add(time.Minute)).
				IssuedAt(time.Now()).
				Build()
			require.NoError(t, err)
			signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkKey))
			require.NoError(t, err)
			return string(signed), token
		},
		wantErr: `failed to fetch JWKS URL`,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			validator := NewDynamicValidator(ctx, "minder")
			token, want := tt.getToken(t)

			got, err := validator.ParseAndValidate(string(token))
			if tt.wantErr != "" {
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Validator.ParseAndValidate() did not return an error matching %q: %s", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validator.ParseAndValidate() returned an error: %v", err)
			}
			if got == nil {
				t.Fatal("Validator.ParseAndValidate() unexpectedly nil")
			}
			if got.Subject() != want.Subject() {
				t.Errorf("Validator.ParseAndValidate() = %s, want %s", got.Subject(), want.Subject())
			}
			if got.Issuer() != want.Issuer() {
				t.Errorf("Validator.ParseAndValidate() = %s, want %s", got.Issuer(), want.Issuer())
			}
			if !slices.Equal(got.Audience(), want.Audience()) {
				t.Errorf("Validator.ParseAndValidate() = %v, want %v", got.Audience(), want.Audience())
			}
		})
	}
}
