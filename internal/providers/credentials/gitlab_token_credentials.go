// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package credentials provides the implementations for the credentials
package credentials

import (
	"net/http"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-containerregistry/pkg/authn"
	"golang.org/x/oauth2"

	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// GitLabTokenCredential is a credential that uses a token
type GitLabTokenCredential struct {
	token string
}

// Ensure that the GitLabTokenCredential implements the GitLabTokenCredential interface
var _ provifv1.GitLabCredential = (*GitLabTokenCredential)(nil)

// NewGitLabTokenCredential creates a new GitLabTokenCredential from the token
func NewGitLabTokenCredential(token string) *GitLabTokenCredential {
	return &GitLabTokenCredential{
		token: token,
	}
}

// SetAuthorizationHeader sets the authorization header on the request
func (t *GitLabTokenCredential) SetAuthorizationHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+t.token)
}

// GetAsContainerAuthenticator returns the token as a container registry authenticator
func (t *GitLabTokenCredential) GetAsContainerAuthenticator(owner string) authn.Authenticator {
	return &authn.Basic{
		Username: owner,
		Password: t.token,
	}
}

// AddToPushOptions adds the credential to the git push options
func (t *GitLabTokenCredential) AddToPushOptions(options *git.PushOptions, owner string) {
	options.Auth = &githttp.BasicAuth{
		Username: owner,
		Password: t.token,
	}
}

// AddToCloneOptions adds the credential to the git clone options
func (t *GitLabTokenCredential) AddToCloneOptions(options *git.CloneOptions) {
	options.Auth = &githttp.BasicAuth{
		// the username can be anything, but it can't be empty
		Username: "minder-user",
		Password: t.token,
	}
}

// GetCacheKey returns the cache key used to look up the REST client
func (t *GitLabTokenCredential) GetCacheKey() string {
	return t.token
}

// GetAsOAuth2TokenSource returns the token as an OAuth2 token source
func (t *GitLabTokenCredential) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: t.token},
	)
}

// GetCredential implements the DirectCredential interface
func (t *GitLabTokenCredential) GetCredential() string {
	return t.token
}
