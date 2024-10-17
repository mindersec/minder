// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"

	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// GitHubInstallationTokenCredential is a credential that uses a GitHub installation access token
type GitHubInstallationTokenCredential struct {
	installationId int64
	token          string
}

// Ensure that the GitHubInstallationTokenCredential implements the GitHubCredential interface
var _ provifv1.GitHubCredential = (*GitHubInstallationTokenCredential)(nil)

// NewGitHubInstallationTokenCredential creates a new GitHubInstallationTokenCredential from the installationId
func NewGitHubInstallationTokenCredential(
	ctx context.Context,
	appId int64,
	privateKey *rsa.PrivateKey,
	endpoint string,
	installationId int64,
) *GitHubInstallationTokenCredential {
	token, err := generateInstallationAccessToken(ctx, appId, privateKey, endpoint, installationId)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).
			Int64("installation_id", installationId).
			Int64("app_id", appId).
			Str("endpoint", endpoint).
			Msg("error generating installation access token")
	}
	return &GitHubInstallationTokenCredential{
		installationId: installationId,
		token:          token,
	}
}

// SetAuthorizationHeader sets the authorization header on the request
func (t *GitHubInstallationTokenCredential) SetAuthorizationHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+t.token)
}

// GetAsContainerAuthenticator returns the token as a container registry authenticator
func (t *GitHubInstallationTokenCredential) GetAsContainerAuthenticator(owner string) authn.Authenticator {
	return &authn.Basic{
		Username: owner,
		Password: t.token,
	}
}

// AddToPushOptions adds the credential to the git push options
func (t *GitHubInstallationTokenCredential) AddToPushOptions(options *git.PushOptions, owner string) {
	options.Auth = &githttp.BasicAuth{
		Username: owner,
		Password: t.token,
	}
}

// AddToCloneOptions adds the credential to the git clone options
func (t *GitHubInstallationTokenCredential) AddToCloneOptions(options *git.CloneOptions) {
	options.Auth = &githttp.BasicAuth{
		// the username can be anything, but it can't be empty
		Username: "minder-user",
		Password: t.token,
	}
}

// GetCacheKey returns the cache key used to look up the REST client
func (t *GitHubInstallationTokenCredential) GetCacheKey() string {
	return strconv.FormatInt(t.installationId, 10)
}

// GetAsOAuth2TokenSource returns the token as an OAuth2 token source
func (t *GitHubInstallationTokenCredential) GetAsOAuth2TokenSource() oauth2.TokenSource {
	// We can use a static token source, since the token expires after 1 hour
	// If we find the token is expiring while in use, we can switch to a ReuseTokenSource
	return oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: t.token},
	)
}

func generateInstallationAccessToken(
	ctx context.Context,
	appId int64,
	privateKey *rsa.PrivateKey,
	endpoint string,
	installationId int64,
) (string, error) {
	jwtToken, err := CreateGitHubAppJWT(appId, privateKey)
	if err != nil {
		return "", fmt.Errorf("unable to create JWT token: %v", err)
	}

	// Use JWT to authenticate with GitHub API for installation access token
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwtToken})

	tc := oauth2.NewClient(ctx, ts)
	appClient := github.NewClient(tc)

	if endpoint != "" {
		parsedURL, err := url.Parse(endpoint)
		if err != nil {
			return "", err
		}
		appClient.BaseURL = parsedURL
	}

	// Create the installation access token
	token, _, err := appClient.Apps.CreateInstallationToken(ctx, installationId, nil)
	if err != nil {
		return "", err
	}
	return token.GetToken(), nil
}
