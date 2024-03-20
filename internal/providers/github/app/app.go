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

// Package app provides the GitHub App specific operations
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	gogithub "github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GithubApp is the string that represents the GitHubApp provider
const GithubApp = "github-app"

// Implements is the list of provider types that the GitHubOAuth provider implements
var Implements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// AuthorizationFlows is the list of authorization flows that the GitHubOAuth provider supports
var AuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowGithubAppFlow,
}

// GitHubAppDelegate is the struct that contains the GitHub App specific operations
type GitHubAppDelegate struct {
	client     *gogithub.Client
	credential provifv1.GitHubCredential
	appId      string
	appName    string
	userId     int64
}

// NewGitHubAppProvider creates a new GitHub App API client
// BaseURL defaults to the public GitHub API, if needing to use a customer domain
// endpoint (as is the case with GitHub Enterprise), set the Endpoint field in
// the GitHubConfig struct
func NewGitHubAppProvider(
	config *minderv1.GitHubAppProviderConfig,
	metrics telemetry.HttpClientMetrics,
	restClientCache ratecache.RestClientCache,
	credential provifv1.GitHubCredential,
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

	oauthDelegate := &GitHubAppDelegate{
		client:     ghClient,
		credential: credential,
		appId:      config.AppId,
		appName:    config.AppName,
		userId:     config.UserId,
	}

	return github.NewGitHub(
		ghClient,
		"",
		restClientCache,
		oauthDelegate,
	), nil
}

// ParseV1Config parses the raw config into a GitHubAppProviderConfig struct
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.GitHubAppProviderConfig, error) {
	type wrapper struct {
		GitHubApp *minderv1.GitHubAppProviderConfig `json:"github-app" yaml:"github-app" mapstructure:"github-app" validate:"required"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.GitHubApp.Validate(); err != nil {
		return nil, fmt.Errorf("error validating GitHubOAuth v1 provider config: %w", err)
	}

	return w.GitHubApp, nil
}

// Ensure that the GitHubAppDelegate client implements the GitHub Delegate interface
var _ github.Delegate = (*GitHubAppDelegate)(nil)

// GetCredential returns the GitHub App installation credential
func (g *GitHubAppDelegate) GetCredential() provifv1.GitHubCredential {
	return g.credential
}

// ListUserRepositories returns a list of repositories for the owner
func (g *GitHubAppDelegate) ListUserRepositories(ctx context.Context, owner string) ([]*minderv1.Repository, error) {
	repos, err := g.ListAllRepositories(ctx, false, owner)
	if err != nil {
		return nil, err
	}

	return github.ConvertRepositories(repos), nil
}

// ListOrganizationRepositories returns a list of repositories for the organization
func (g *GitHubAppDelegate) ListOrganizationRepositories(
	ctx context.Context,
	owner string,
) ([]*minderv1.Repository, error) {
	repos, err := g.ListAllRepositories(ctx, true, owner)
	if err != nil {
		return nil, err
	}

	return github.ConvertRepositories(repos), nil
}

// ListAllRepositories returns a list of all repositories accessible to the GitHub App installation
func (g *GitHubAppDelegate) ListAllRepositories(ctx context.Context, _ bool, _ string) ([]*gogithub.Repository, error) {
	listOpt := &gogithub.ListOptions{
		PerPage: 100,
	}

	// create a slice to hold the repositories
	var allRepos []*gogithub.Repository
	for {
		var repos *gogithub.ListRepositories
		var resp *gogithub.Response
		var err error

		repos, resp, err = g.client.Apps.ListRepos(ctx, listOpt)

		if err != nil {
			return allRepos, err
		}
		allRepos = append(allRepos, repos.Repositories...)
		if resp.NextPage == 0 {
			break
		}

		listOpt.Page = resp.NextPage
	}

	return allRepos, nil
}

// GetUserId returns the user id for the GitHub App user
func (g *GitHubAppDelegate) GetUserId(_ context.Context) (int64, error) {
	// note: this is different from the App ID
	return g.userId, nil
}

// GetName returns the username for the GitHub App user
func (g *GitHubAppDelegate) GetName(_ context.Context) (string, error) {
	return fmt.Sprintf("%s[bot]", g.appName), nil
}

// GetLogin returns the username for the GitHub App user
func (g *GitHubAppDelegate) GetLogin(ctx context.Context) (string, error) {
	return g.GetName(ctx)
}

// GetPrimaryEmail returns the email for the GitHub App user
func (g *GitHubAppDelegate) GetPrimaryEmail(ctx context.Context) (string, error) {
	userId, err := g.GetUserId(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting user ID: %v", err)
	}
	return fmt.Sprintf("%d+github-actions[bot]@users.noreply.github.com", userId), nil
}
