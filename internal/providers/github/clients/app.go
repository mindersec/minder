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

package clients

import (
	"context"
	"encoding/json"
	"fmt"

	gogithub "github.com/google/go-github/v61/github"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/github"
	ghcommon "github.com/stacklok/minder/internal/providers/github/common"
	"github.com/stacklok/minder/internal/providers/ratecache"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GithubApp is the string that represents the GitHubApp provider
const GithubApp = "github-app"

// AppImplements is the list of provider types that the GitHubOAuth provider implements
var AppImplements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// AppAuthorizationFlows is the list of authorization flows that the GitHubOAuth provider supports
var AppAuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowGithubAppFlow,
}

// GitHubAppDelegate is the struct that contains the GitHub App specific operations
type GitHubAppDelegate struct {
	client        *gogithub.Client
	credential    provifv1.GitHubCredential
	appName       string
	defaultUserId int64
	isOrg         bool
}

// NewAppDelegate creates a GitHubOAuthDelegate from a GitHub client
// This exists as a separate function to allow the provider creation code
// to use its methods without instantiating a full provider.
func NewAppDelegate(
	client *gogithub.Client,
	credential provifv1.GitHubCredential,
	appName string,
	defaultUserId int64,
	isOrg bool,
) *GitHubAppDelegate {
	return &GitHubAppDelegate{
		client:        client,
		credential:    credential,
		appName:       appName,
		defaultUserId: defaultUserId,
		isOrg:         isOrg,
	}
}

// NewGitHubAppProvider creates a new GitHub App API client
// BaseURL defaults to the public GitHub API, if needing to use a customer domain
// endpoint (as is the case with GitHub Enterprise), set the Endpoint field in
// the GitHubConfig struct
func NewGitHubAppProvider(
	cfg *minderv1.GitHubAppProviderConfig,
	appConfig *server.GitHubAppConfig,
	restClientCache ratecache.RestClientCache,
	credential provifv1.GitHubCredential,
	packageListingClient *gogithub.Client,
	ghClientFactory GitHubClientFactory,
	isOrg bool,
) (*github.GitHub, error) {
	appName := appConfig.AppName
	userId := appConfig.UserID

	ghClient, delegate, err := ghClientFactory.BuildAppClient(
		cfg.Endpoint,
		credential,
		appName,
		userId,
		isOrg,
	)
	if err != nil {
		return nil, err
	}

	return github.NewGitHub(
		ghClient,
		// Use the fallback token for package listing, since fine-grained tokens don't have access
		packageListingClient,
		restClientCache,
		delegate,
	), nil
}

// ParseV1AppConfig parses the raw config into a GitHubAppProviderConfig struct
func ParseV1AppConfig(rawCfg json.RawMessage) (
	*minderv1.ProviderConfig,
	*minderv1.GitHubAppProviderConfig,
	error,
) {
	// embedding the struct to expose its JSON tags
	type wrapper struct {
		*minderv1.ProviderConfig
		GitHubApp *minderv1.GitHubAppProviderConfig `json:"github-app" yaml:"github-app" mapstructure:"github-app" validate:"required"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.GitHubApp.Validate(); err != nil {
		return nil, nil, fmt.Errorf("error validating GitHubApp v1 provider config: %w", err)
	}

	if w.ProviderConfig != nil {
		if err := w.ProviderConfig.Validate(); err != nil {
			return nil, nil, fmt.Errorf("error validating provider config: %w", err)
		}
	}

	return w.ProviderConfig, w.GitHubApp, nil
}

// Ensure that the GitHubAppDelegate client implements the GitHub Delegate interface
var _ github.Delegate = (*GitHubAppDelegate)(nil)

// GetCredential returns the GitHub App installation credential
func (g *GitHubAppDelegate) GetCredential() provifv1.GitHubCredential {
	return g.credential
}

// GetOwner returns the owner filter
func (_ *GitHubAppDelegate) GetOwner() string {
	return ""
}

// IsOrg returns true if the owner is an organization
func (g *GitHubAppDelegate) IsOrg() bool {
	return g.isOrg
}

// ListAllRepositories returns a list of all repositories accessible to the GitHub App installation
func (g *GitHubAppDelegate) ListAllRepositories(ctx context.Context) ([]*minderv1.Repository, error) {
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
			return ghcommon.ConvertRepositories(allRepos), fmt.Errorf("error listing repositories: %w", err)
		}
		allRepos = append(allRepos, repos.Repositories...)
		if resp.NextPage == 0 {
			break
		}

		listOpt.Page = resp.NextPage
	}

	return ghcommon.ConvertRepositories(allRepos), nil
}

// GetUserId returns the user id for the GitHub App user
func (g *GitHubAppDelegate) GetUserId(ctx context.Context) (int64, error) {
	// Try to get this user ID from the GitHub API
	user, _, err := g.client.Users.Get(ctx, "")
	if err != nil {
		// Fallback to the configured user ID
		// note: this is different from the App ID
		return g.defaultUserId, nil
	}
	return user.GetID(), nil
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
