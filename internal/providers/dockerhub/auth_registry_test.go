// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/providers/credentials"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	testNamespace = "acme-org"
	testPAT       = "dckr_pat_test"
	testJWT       = "short-lived-registry-jwt"
)

func newTokenAuthRegistry(t *testing.T, inner http.Handler) *httptest.Server {
	t.Helper()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			user, pass, ok := req.BasicAuth()
			if !ok || user != testNamespace || pass != testPAT {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"token": testJWT})
			return
		}

		if req.Header.Get("Authorization") != "Bearer "+testJWT {
			w.Header().Set("WWW-Authenticate",
				fmt.Sprintf(`Bearer realm="%s/token",service="registry.test"`, srv.URL))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		inner.ServeHTTP(w, req)
	}))
	t.Cleanup(srv.Close)

	return srv
}

func seedImage(t *testing.T, inner http.Handler, repoTag string) {
	t.Helper()

	seed := httptest.NewServer(inner)
	t.Cleanup(seed.Close)

	ref, err := name.ParseReference(strings.TrimPrefix(seed.URL, "http://")+"/"+repoTag, name.Insecure)
	require.NoError(t, err)
	img, err := random.Image(64, 1)
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, img))
}

func TestGetAuthenticatorExchangesBasicForRegistryToken(t *testing.T) {
	t.Parallel()

	inner := registry.New(registry.Logger(log.New(io.Discard, "", 0)))
	repoTag := testNamespace + "/app:v1"
	seedImage(t, inner, repoTag)
	reg := newTokenAuthRegistry(t, inner)

	ref, err := name.ParseReference(strings.TrimPrefix(reg.URL, "http://")+"/"+repoTag, name.Insecure)
	require.NoError(t, err)

	t.Run("dockerhub authenticator succeeds via token exchange", func(t *testing.T) {
		t.Parallel()

		lister, err := New(credentials.NewOAuth2TokenCredential(testPAT),
			&minderv1.DockerHubProviderConfig{Namespace: proto.String(testNamespace)})
		require.NoError(t, err)

		auth, err := lister.GetAuthenticator()
		require.NoError(t, err)
		assert.Equal(t, &authn.Basic{Username: testNamespace, Password: testPAT}, auth)

		_, err = remote.Get(ref, remote.WithAuth(auth), remote.WithContext(context.Background()))
		require.NoError(t, err)
	})

	t.Run("raw PAT as bearer is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := remote.Get(ref, remote.WithAuth(&authn.Bearer{Token: testPAT}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("wrong username is rejected by token server", func(t *testing.T) {
		t.Parallel()

		_, err := remote.Get(ref, remote.WithAuth(&authn.Basic{Username: "someone-else", Password: testPAT}))
		require.Error(t, err)
	})
}

func TestGetAuthenticatorNilCredential(t *testing.T) {
	t.Parallel()

	lister := &dockerHubImageLister{namespace: testNamespace}
	auth, err := lister.GetAuthenticator()
	require.NoError(t, err)
	assert.Equal(t, authn.Anonymous, auth)
}
