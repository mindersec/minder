// Copyright 2023 Stacklok, Inc
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

// Package oauth provides a client for interacting with the GitHub API using OAuth 2.0 authorization
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	gogithub "github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/github"
	ghcommon "github.com/stacklok/minder/internal/providers/github/common"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// Github is the string that represents the GitHubOAuth provider
	Github = "github"
	// GithubApp is the string that represents the GitHub App provider
	GithubApp = "github-app"
)

// Implements is the list of provider types that the GitHubOAuth provider implements
var Implements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// AuthorizationFlows is the list of authorization flows that the GitHubOAuth provider supports
var AuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowUserInput,
	db.AuthorizationFlowOauth2AuthorizationCodeFlow,
}

// GitHubOAuthDelegate is the struct that contains the GitHub access token specifc operations
type GitHubOAuthDelegate struct {
	client     *gogithub.Client
	credential provifv1.GitHubCredential
	owner      string
}

// Ensure that the GitHubOAuthDelegate client implements the GitHub Delegate interface
var _ github.Delegate = (*GitHubOAuthDelegate)(nil)

// NewRestClient creates a new GitHub REST API client
// BaseURL defaults to the public GitHub API, if needing to use a customer domain
// endpoint (as is the case with GitHub Enterprise), set the Endpoint field in
// the GitHubConfig struct
func NewRestClient(
	config *minderv1.GitHubProviderConfig,
	metrics telemetry.HttpClientMetrics,
	restClientCache ratecache.RestClientCache,
	credential provifv1.GitHubCredential,
	owner string,
) (*github.GitHub, error) {
	var err error

	tc := &http.Client{
		Transport: &oauth2.Transport{
			Base:   http.DefaultClient.Transport,
			Source: credential.GetAsOAuth2TokenSource(),
		},
	}

	tc.Transport, err = metrics.NewDurationRoundTripper(tc.Transport, db.ProviderTypeGithub)
	if err != nil {
		return nil, fmt.Errorf("error creating duration round tripper: %w", err)
	}

	ghClient := gogithub.NewClient(tc)

	if config.Endpoint != "" {
		parsedURL, err := url.Parse(config.Endpoint)
		if err != nil {
			return nil, err
		}
		ghClient.BaseURL = parsedURL
	}

	oauthDelegate := &GitHubOAuthDelegate{
		client:     ghClient,
		credential: credential,
		owner:      owner,
	}

	return github.NewGitHub(
		ghClient,
		ghClient, // use the same client for listing packages and all other operations
		restClientCache,
		oauthDelegate,
	), nil
}

// ParseV1Config parses the raw config into a GitHubConfig struct
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.GitHubProviderConfig, error) {
	type wrapper struct {
		GitHub *minderv1.GitHubProviderConfig `json:"github" yaml:"github" mapstructure:"github" validate:"required"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.GitHub.Validate(); err != nil {
		return nil, fmt.Errorf("error validating GitHubOAuth v1 provider config: %w", err)
	}

	return w.GitHub, nil
}

// GetCredential returns the GitHub OAuth credential
func (o *GitHubOAuthDelegate) GetCredential() provifv1.GitHubCredential {
	return o.credential
}

// GetOwner returns the owner filter
func (o *GitHubOAuthDelegate) GetOwner() string {
	return o.owner
}

// IsOrg returns true if the owner is an organization
func (o *GitHubOAuthDelegate) IsOrg() bool {
	return o.owner != ""
}

// ListAllRepositories returns a list of all repositories for the authenticated user
// Two APIs are available, contigent on whether the token is for a user or an organization
func (o *GitHubOAuthDelegate) ListAllRepositories(ctx context.Context) ([]*minderv1.Repository, error) {
	opt := &gogithub.RepositoryListByAuthenticatedUserOptions{
		ListOptions: gogithub.ListOptions{
			PerPage: 100,
		},
		Affiliation: "owner",
	}

	orgOpt := &gogithub.RepositoryListByOrgOptions{
		ListOptions: gogithub.ListOptions{
			PerPage: 100,
		},
	}

	// create a slice to hold the repositories
	var allRepos []*gogithub.Repository
	for {
		var repos []*gogithub.Repository
		var resp *gogithub.Response
		var err error

		if o.owner != "" {
			repos, resp, err = o.client.Repositories.ListByOrg(ctx, o.owner, orgOpt)
		} else {
			repos, resp, err = o.client.Repositories.ListByAuthenticatedUser(ctx, opt)
		}

		if err != nil {
			return ghcommon.ConvertRepositories(allRepos), fmt.Errorf("error listing repositories: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}

		if o.owner != "" {
			orgOpt.Page = resp.NextPage
		} else {
			opt.Page = resp.NextPage
		}
	}

	return ghcommon.ConvertRepositories(allRepos), nil
}

// GetUserId returns the user id for the authenticated user
func (o *GitHubOAuthDelegate) GetUserId(ctx context.Context) (int64, error) {
	user, _, err := o.client.Users.Get(ctx, "")
	if err != nil {
		return 0, err
	}
	return user.GetID(), nil
}

// GetName returns the username for the authenticated user
func (o *GitHubOAuthDelegate) GetName(ctx context.Context) (string, error) {
	user, _, err := o.client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetName(), nil
}

// GetLogin returns the login for the authenticated user
func (o *GitHubOAuthDelegate) GetLogin(ctx context.Context) (string, error) {
	user, _, err := o.client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}

// GetPrimaryEmail returns the primary email for the authenticated user.
func (o *GitHubOAuthDelegate) GetPrimaryEmail(ctx context.Context) (string, error) {
	emails, _, err := o.client.Users.ListEmails(ctx, &gogithub.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("cannot get email: %w", err)
	}

	fallback := ""
	for _, email := range emails {
		if fallback == "" {
			fallback = email.GetEmail()
		}
		if email.GetPrimary() {
			return email.GetEmail(), nil
		}
	}

	return fallback, nil
}
