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
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/lestrrat-go/jwx/v3/jwt/openid"
	"github.com/stretchr/testify/require"
)

// singleIssuer serves one issuer's discovery doc and JWKS and counts requests to each.
type singleIssuer struct {
	issuer         string
	token          string
	discoveryCalls atomic.Int64
	jwksCalls      atomic.Int64
	// failDiscoveries makes the first N discovery requests return 500.
	failDiscoveries atomic.Int64
}

func newSingleIssuer(t *testing.T) *singleIssuer {
	t.Helper()

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

	s := &singleIssuer{}
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	s.issuer = server.URL

	mux.HandleFunc("/certs", func(w http.ResponseWriter, _ *http.Request) {
		s.jwksCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(keySetJSON)
	})
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		if s.discoveryCalls.Add(1) <= s.failDiscoveries.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"issuer":%q,"jwks_uri":"%s/certs"}`, s.issuer, s.issuer)
	})

	token, err := openid.NewBuilder().
		Issuer(s.issuer).
		Subject("test").
		Audience([]string{"minder"}).
		Expiration(time.Now().Add(time.Minute)).
		IssuedAt(time.Now()).
		Build()
	require.NoError(t, err)
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), jwkKey))
	require.NoError(t, err)
	s.token = string(signed)

	return s
}

// TestValidator_ConcurrentSameIssuer sends several valid tokens from the same,
// not-yet-seen issuer at once. Every one of them should be accepted.
func TestValidator_ConcurrentSameIssuer(t *testing.T) {
	t.Parallel()

	const numRequests = 8

	iss := newSingleIssuer(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator := NewDynamicValidator(ctx, "minder", []string{iss.issuer})
	require.NotNil(t, validator)

	start := make(chan struct{})
	errs := make(chan error, numRequests)
	var wg sync.WaitGroup
	for range numRequests {
		wg.Go(func() {
			<-start
			_, err := validator.ParseAndValidate(iss.token)
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)

	failed := 0
	for err := range errs {
		if err != nil {
			failed++
			t.Logf("valid token rejected: %v", err)
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d valid tokens rejected", failed, numRequests)
	}
}

// TestValidator_DiscoveryFetchedOnce validates the same token several times in
// a row and checks the issuer's discovery doc is only fetched once.
func TestValidator_DiscoveryFetchedOnce(t *testing.T) {
	t.Parallel()

	iss := newSingleIssuer(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator := NewDynamicValidator(ctx, "minder", []string{iss.issuer})
	require.NotNil(t, validator)

	_, err := validator.ParseAndValidate(iss.token)
	require.NoError(t, err)
	for range 5 {
		_, err := validator.ParseAndValidate(iss.token)
		require.NoError(t, err)
	}

	require.Equal(t, int64(1), iss.discoveryCalls.Load(),
		"discovery doc fetched %d times, JWKS fetched %d times, after 6 validations",
		iss.discoveryCalls.Load(), iss.jwksCalls.Load())
}

// TestValidator_DiscoveryRetriesAfterFailure checks that a failed discovery is
// not cached, so the next request retries it, and that a successful one is.
func TestValidator_DiscoveryRetriesAfterFailure(t *testing.T) {
	t.Parallel()

	iss := newSingleIssuer(t)
	iss.failDiscoveries.Store(1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator := NewDynamicValidator(ctx, "minder", []string{iss.issuer})
	require.NotNil(t, validator)

	_, err := validator.ParseAndValidate(iss.token)
	require.Error(t, err, "first validation should fail while discovery returns 500")

	_, err = validator.ParseAndValidate(iss.token)
	require.NoError(t, err, "second validation should retry discovery and succeed")
	require.Equal(t, int64(2), iss.discoveryCalls.Load(),
		"discovery should be retried after a failure")

	_, err = validator.ParseAndValidate(iss.token)
	require.NoError(t, err)
	require.Equal(t, int64(2), iss.discoveryCalls.Load(),
		"discovery should be cached after it succeeds")
}
