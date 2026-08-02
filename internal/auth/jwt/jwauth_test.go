// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidate(t *testing.T) {
	t.Parallel()

	jwks := jwk.NewSet()
	privateKey, publicKey := randomKeypair(2048)
	privateJwk, _ := jwk.Import(privateKey)
	err := privateJwk.Set(jwk.KeyIDKey, `mykey`)
	require.NoError(t, err, "failed to setup private key ID")

	publicJwk, _ := jwk.Import(publicKey)
	err = publicJwk.Set(jwk.KeyIDKey, "mykey")
	require.NoError(t, err, "failed to setup public key ID")
	err = publicJwk.Set(jwk.AlgorithmKey, jwa.RS256().String())
	require.NoError(t, err, "failed to setup public key algorithm")

	err = jwks.AddKey(publicJwk)
	require.NoError(t, err, "failed to setup JWK set")
	keySetJSON, err := json.Marshal(jwks)
	require.NoError(t, err, "failed to marshal JWK set")

	issUrl := "https://localhost/realm/foo"

	mux := http.NewServeMux()
	mux.HandleFunc("/certs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(keySetJSON)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	testCases := []struct {
		name       string
		buildToken func() string
		wantErr    string
	}{
		{
			name: "Valid token",
			buildToken: func() string {
				token, _ := jwtBuilder("123", issUrl, "minder").Expiration(time.Now().Add(time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privateJwk))
				return string(signed)
			},
		},
		{
			name: "Expired token",
			buildToken: func() string {
				token, _ := jwtBuilder("123", issUrl, "minder").Expiration(time.Now().Add(-time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privateJwk))
				return string(signed)
			},
			wantErr: "token is expired",
		},
		{
			name: "Invalid signature",
			buildToken: func() string {
				otherKey, _ := randomKeypair(2048)
				otherJwk, _ := jwk.Import(otherKey)
				err = otherJwk.Set(jwk.KeyIDKey, `otherKey`)
				require.NoError(t, err, "failed to setup signing key ID")
				token, _ := jwtBuilder("123", issUrl, "minder").Expiration(time.Now().Add(time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256(), otherJwk))
				return string(signed)
			},
			wantErr: "could not verify message using any of the signatures or keys",
		},
		{
			name: "Invalid token",
			buildToken: func() string {
				return "invalid"
			},
			wantErr: "failed to parse string: unknown payload type (payload is not JWT?)",
		},
		{
			name: "Missing subject claim",
			buildToken: func() string {
				token, _ := jwtBuilder("", issUrl, "minder").Expiration(time.Now().Add(time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privateJwk))
				return string(signed)
			},
			wantErr: "provided token is missing required subject claim",
		},
		{
			name: "Missing issuer claim",
			buildToken: func() string {
				token, _ := jwtBuilder("123", "", "minder").Expiration(time.Now().Add(time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privateJwk))
				return string(signed)
			},
			wantErr: `field "iss" not found`,
		},
		{
			name: "Missing audience claim",
			buildToken: func() string {
				token, _ := jwtBuilder("123", issUrl, "").Expiration(time.Now().Add(time.Duration(1) * time.Minute)).Build()
				signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privateJwk))
				return string(signed)
			},
			wantErr: `field "aud" not found`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			jwtValidator, err := NewJwtValidator(context.Background(), server.URL+"/certs", issUrl, "minder")
			require.NoError(t, err)
			token, err := jwtValidator.ParseAndValidate(tc.buildToken())
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				_, err := GetUserEmailFromContext(context.Background())
				require.Error(t, err)
				_, ok := GetUserClaimFromContext[string](context.Background(), "sub")
				require.False(t, ok)

				return
			}

			require.NoError(t, err)
			ctx := WithAuthTokenContext(context.Background(), token)

			// We only have one happy path at the moment, so these are hard-coded.
			ctxToken, err := GetUserTokenFromContext(ctx)
			require.NoError(t, err)
			assert.Equal(t, token, ctxToken)

			id, ok := GetUserClaimFromContext[string](ctx, "sub")
			require.True(t, ok)
			assert.Equal(t, "123", id)

			email, err := GetUserEmailFromContext(ctx)
			require.NoError(t, err)
			assert.Equal(t, "bob@example.com", email)
		})
	}
}

// RandomKeypair returns a random RSA keypair
func randomKeypair(length int) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(crand.Reader, length)
	if err != nil {
		return nil, nil
	}
	publicKey := &privateKey.PublicKey

	return privateKey, publicKey
}

func jwtBuilder(sub, iss, aud string) *jwt.Builder {
	r := jwt.NewBuilder()

	if sub != "" {
		r = r.Subject(sub)
	}
	if iss != "" {
		r = r.Issuer(iss)
	}
	if aud != "" {
		r = r.Audience([]string{aud})
	}
	r.Claim("email", "bob@example.com")

	return r
}
