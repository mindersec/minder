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
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GitHubInstallationTokenCredential is a credential that uses a GitHub installation access token
type GitHubInstallationTokenCredential struct {
	installationId string
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
	installationId string,
) *GitHubInstallationTokenCredential {
	token, err := generateInstallationAccessToken(ctx, appId, privateKey, endpoint, installationId)
	if err != nil {
		fmt.Printf("error generating installation access token: %v", err)
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
	return t.installationId
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
	installationId string,
) (string, error) {
	installationIdInt, err := strconv.ParseInt(installationId, 10, 64)
	if err != nil {
		return "", fmt.Errorf("unable to parse installationId to integer: %v", err)
	}

	jwtToken, err := createJWT(appId, privateKey)
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
	token, _, err := appClient.Apps.CreateInstallationToken(ctx, installationIdInt, nil)
	if err != nil {
		return "", err
	}
	return token.GetToken(), nil
}

func createJWT(appId int64, privateKey *rsa.PrivateKey) (string, error) {
	// Create the Claims
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 10).Unix(),
		"iss": appId,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	jwtToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("unable to sign JWT token: %v", err)
	}

	return jwtToken, nil
}
