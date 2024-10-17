// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

var (
	generatedToken               = "test_token"
	installationId               = int64(987654)
	gitHubInstallationCredential = GitHubInstallationTokenCredential{
		installationId: installationId,
		token:          generatedToken,
	}
)

func TestNewGitHubInstallationTokenCredential(t *testing.T) {
	t.Parallel()

	installationTokensEndpoint := fmt.Sprintf("/app/installations/%v/access_tokens", installationId)
	expectedToken := "some-token"

	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		t.Fatal("Unable to generate private key")
	}

	mockGithubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case installationTokensEndpoint:
			data := github.InstallationToken{
				Token: &expectedToken,
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			err := json.NewEncoder(w).Encode(data)
			if err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("Unexpected call to mock server endpoint %s", r.URL.Path)
		}
	}))
	defer mockGithubServer.Close()

	credential := NewGitHubInstallationTokenCredential(context.Background(), 123456, privateKey, mockGithubServer.URL+"/", installationId)

	assert.Equal(t, expectedToken, credential.token)
	assert.Equal(t, installationId, credential.installationId)
}

func TestGitHubInstallationTokenCredentialSetAuthorizationHeader(t *testing.T) {
	t.Parallel()

	expected := "Bearer test_token"
	req := &http.Request{
		Header: http.Header{},
	}
	gitHubInstallationCredential.SetAuthorizationHeader(req)
	require.Equal(t, expected, req.Header.Get("Authorization"))
}

func TestGitHubInstallationTokenCredentialAddToPushOptions(t *testing.T) {
	t.Parallel()

	username := "test_username"
	expected := &githttp.BasicAuth{
		Username: username,
		Password: token,
	}
	pushOptions := &git.PushOptions{}
	gitHubInstallationCredential.AddToPushOptions(pushOptions, username)
	require.Equal(t, expected, pushOptions.Auth)
}

func TestGitHubInstallationTokenCredentialAddToClone(t *testing.T) {
	t.Parallel()

	expected := &githttp.BasicAuth{
		Username: "minder-user",
		Password: token,
	}
	cloneOptions := &git.CloneOptions{}
	gitHubInstallationCredential.AddToCloneOptions(cloneOptions)
	require.Equal(t, expected, cloneOptions.Auth)
}

func TestGitHubInstallationTokenCredentialGetAsContainerAuthenticator(t *testing.T) {
	t.Parallel()

	username := "test_username"
	expected := &authn.Basic{
		Username: username,
		Password: token,
	}
	require.Equal(t, expected, gitHubInstallationCredential.GetAsContainerAuthenticator(username))
}

func TestGitHubInstallationTokenCredentialGetAsOAuth2TokenSource(t *testing.T) {
	t.Parallel()

	expected := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	require.Equal(t, expected, gitHubInstallationCredential.GetAsOAuth2TokenSource())
}

func TestGitHubInstallationTokenCredentialGetCacheKey(t *testing.T) {
	t.Parallel()

	// we still want to compare against string(ID) because the cache key is supposed to return a string
	require.Equal(t, fmt.Sprint(installationId), gitHubInstallationCredential.GetCacheKey())
}
