// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitLabTokenCredential(t *testing.T) {
	t.Parallel()

	testToken := "glpat-test-token-12345"
	cred := NewGitLabTokenCredential(testToken)
	require.NotNil(t, cred)

	t.Run("GetCredential", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, testToken, cred.GetCredential())
	})

	t.Run("GetCacheKey", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, testToken, cred.GetCacheKey())
	})

	t.Run("SetAuthorizationHeader", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://gitlab.com/api/v4/projects", nil)
		require.NoError(t, err)

		cred.SetAuthorizationHeader(req)
		assert.Equal(t, "Bearer "+testToken, req.Header.Get("Authorization"))
	})

	t.Run("GetAsContainerAuthenticator", func(t *testing.T) {
		t.Parallel()

		owner := "gitlab-org"
		auth := cred.GetAsContainerAuthenticator(owner)
		require.NotNil(t, auth)

		basicAuth, ok := auth.(*authn.Basic)
		require.True(t, ok)
		assert.Equal(t, owner, basicAuth.Username)
		assert.Equal(t, testToken, basicAuth.Password)
	})

	t.Run("AddToPushOptions", func(t *testing.T) {
		t.Parallel()

		owner := "gitlab-org"
		pushOpts := &git.PushOptions{}
		cred.AddToPushOptions(pushOpts, owner)

		basicAuth, ok := pushOpts.Auth.(*githttp.BasicAuth)
		require.True(t, ok)
		assert.Equal(t, owner, basicAuth.Username)
		assert.Equal(t, testToken, basicAuth.Password)
	})

	t.Run("AddToCloneOptions", func(t *testing.T) {
		t.Parallel()

		cloneOpts := &git.CloneOptions{}
		cred.AddToCloneOptions(cloneOpts)

		basicAuth, ok := cloneOpts.Auth.(*githttp.BasicAuth)
		require.True(t, ok)
		assert.Equal(t, "minder-user", basicAuth.Username)
		assert.Equal(t, testToken, basicAuth.Password)
	})

	t.Run("GetAsOAuth2TokenSource", func(t *testing.T) {
		t.Parallel()

		ts := cred.GetAsOAuth2TokenSource()
		require.NotNil(t, ts)

		tok, err := ts.Token()
		require.NoError(t, err)
		assert.Equal(t, testToken, tok.AccessToken)
	})
}
