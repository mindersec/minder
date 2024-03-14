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

// Package credentials provides the implementations for the credentials
package credentials

import (
	"net/http"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-containerregistry/pkg/authn"
	"golang.org/x/oauth2"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GitHubTokenCredential is a credential that uses a token
type GitHubTokenCredential struct {
	token string
}

// Ensure that the GitHubTokenCredential implements the GitHubTokenCredential interface
var _ provifv1.GitHubCredential = (*GitHubTokenCredential)(nil)

// NewGitHubTokenCredential creates a new GitHubTokenCredential from the token
func NewGitHubTokenCredential(token string) *GitHubTokenCredential {
	return &GitHubTokenCredential{
		token: token,
	}
}

// SetAuthorizationHeader sets the authorization header on the request
func (t *GitHubTokenCredential) SetAuthorizationHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+t.token)
}

// GetAsContainerAuthenticator returns the token as a container registry authenticator
func (t *GitHubTokenCredential) GetAsContainerAuthenticator(owner string) authn.Authenticator {
	return &authn.Basic{
		Username: owner,
		Password: t.token,
	}
}

// AddToPushOptions adds the credential to the git push options
func (t *GitHubTokenCredential) AddToPushOptions(options *git.PushOptions, owner string) {
	options.Auth = &githttp.BasicAuth{
		Username: owner,
		Password: t.token,
	}
}

// AddToCloneOptions adds the credential to the git clone options
func (t *GitHubTokenCredential) AddToCloneOptions(options *git.CloneOptions) {
	options.Auth = &githttp.BasicAuth{
		// the username can be anything, but it can't be empty
		Username: "minder-user",
		Password: t.token,
	}
}

// GetCacheKey returns the cache key used to look up the REST client
func (t *GitHubTokenCredential) GetCacheKey() string {
	return t.token
}

// GetAsOAuth2TokenSource returns the token as an OAuth2 token source
func (t *GitHubTokenCredential) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: t.token},
	)
}
