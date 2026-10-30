// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dynamic

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/lestrrat-go/jwx/v3/jwt/openid"
	"github.com/stretchr/testify/require"
)

// TestValidator_ConcurrentFirstUse validates tokens from several allowed
// issuers, each with its own JWKS URL, concurrently on one shared Validator.
// This is what the gRPC interceptor does when requests from issuers not seen
// yet arrive at the same time.
func TestValidator_ConcurrentFirstUse(t *testing.T) {
	t.Parallel()

	const numIssuers = 8

	keyGen := rand.New(rand.NewSource(12345))
	key, err := rsa.GenerateKey(keyGen, 2048)
	require.NoError(t, err)
	jwkKey, err := jwk.Import(key)
	require.NoError(t, err)
	require.NoError(t, jwkKey.Set(jwk.KeyIDKey, "test"))
	require.NoError(t, jwkKey.Set(jwk.AlgorithmKey, jwa.RS256().String()))
	require.NoError(t, jwkKey.Set(jwk.KeyUsageKey, "sig"))
	pubKey, err := jwkKey.PublicKey()
	require.NoError(t, err)

	keySet := jwk.NewSet()
	require.NoError(t, keySet.AddKey(pubKey))
	keySetJSON, err := json.Marshal(keySet)
	require.NoError(t, err)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	issuers := make([]string, 0, numIssuers)
	for i := range numIssuers {
		issuer := fmt.Sprintf("%s/iss%d", server.URL, i)
		issuers = append(issuers, issuer)
		mux.HandleFunc(fmt.Sprintf("/iss%d/certs", i), func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(keySetJSON)
		})
		mux.HandleFunc(fmt.Sprintf("/iss%d/.well-known/openid-configuration", i),
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"issuer":%q,"jwks_uri":"%s/certs"}`, issuer, issuer)
			})
	}

	tokens := make([]string, 0, numIssuers)
	for _, issuer := range issuers {
		token, err := openid.NewBuilder().
			Issuer(issuer).
			Subject("test").
			Audience([]string{"minder"}).
			Expiration(time.Now().Add(time.Minute)).
			IssuedAt(time.Now()).
			Build()
		require.NoError(t, err)
		signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), jwkKey))
		require.NoError(t, err)
		tokens = append(tokens, string(signed))
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator := NewDynamicValidator(ctx, "minder", issuers)
	require.NotNil(t, validator)

	start := make(chan struct{})
	errs := make(chan error, numIssuers)
	var wg sync.WaitGroup
	for _, tok := range tokens {
		wg.Go(func() {
			<-start
			_, err := validator.ParseAndValidate(tok)
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
}
