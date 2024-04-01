// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credentials

import (
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

var (
	token      = "test_token"
	credential = NewGitHubTokenCredential(token)
)

func TestGitHubTokenCredentialSetAuthorizationHeader(t *testing.T) {
	t.Parallel()

	expected := "Bearer test_token"
	req := &http.Request{
		Header: http.Header{},
	}
	credential.SetAuthorizationHeader(req)
	require.Equal(t, expected, req.Header.Get("Authorization"))
}

func TestGitHubTokenCredentialAddToPushOptions(t *testing.T) {
	t.Parallel()

	username := "test_user"
	expected := &githttp.BasicAuth{
		Username: username,
		Password: token,
	}
	pushOptions := &git.PushOptions{}
	credential.AddToPushOptions(pushOptions, username)
	require.Equal(t, expected, pushOptions.Auth)
}

func TestGitHubTokenCredentialAddToClone(t *testing.T) {
	t.Parallel()

	expected := &githttp.BasicAuth{
		Username: "minder-user",
		Password: token,
	}
	cloneOptions := &git.CloneOptions{}
	credential.AddToCloneOptions(cloneOptions)
	require.Equal(t, expected, cloneOptions.Auth)
}

func TestGitHubTokenCredentialGetAsContainerAuthenticator(t *testing.T) {
	t.Parallel()

	username := "test_user"
	expected := &authn.Basic{
		Username: username,
		Password: token,
	}
	require.Equal(t, expected, credential.GetAsContainerAuthenticator(username))
}

func TestGitHubTokenCredentialGetAsOAuth2TokenSource(t *testing.T) {
	t.Parallel()

	expected := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	require.Equal(t, expected, credential.GetAsOAuth2TokenSource())
}

func TestGitHubTokenCredentialGetCacheKey(t *testing.T) {
	t.Parallel()

	require.Equal(t, token, credential.GetCacheKey())
}
