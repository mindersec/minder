// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"encoding/json"
	"fmt"

	"dario.cat/mergo"
	"github.com/go-playground/validator/v10"
	gogithub "github.com/google/go-github/v63/github"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/github"
	ghcommon "github.com/mindersec/minder/internal/providers/github/common"
	"github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/ratecache"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/server"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
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
	appConfig *server.ProviderConfig,
	whcfg *server.WebhookConfig,
	restClientCache ratecache.RestClientCache,
	credential provifv1.GitHubCredential,
	packageListingClient *gogithub.Client,
	ghClientFactory GitHubClientFactory,
	propertyFetchers properties.GhPropertyFetcherFactory,
	isOrg bool,
) (*github.GitHub, error) {
	if appConfig == nil || appConfig.GitHubApp == nil {
		return nil, fmt.Errorf("missing GitHub App configuration")
	}
	appName := appConfig.GitHubApp.AppName
	userId := appConfig.GitHubApp.UserID

	ghClient, delegate, err := ghClientFactory.BuildAppClient(
		cfg.GetEndpoint(),
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
		appConfig,
		whcfg,
		propertyFetchers,
	), nil
}

// embedding the struct to expose its JSON tags
type appConfigWrapper struct {
	*minderv1.ProviderConfig
	GitHubAppOldKey *minderv1.GitHubAppProviderConfig `json:"github-app" yaml:"github-app" mapstructure:"github-app"`
	GitHubApp       *minderv1.GitHubAppProviderConfig `json:"github_app" yaml:"github_app" mapstructure:"github_app"`
}

func getDefaultAppConfig() appConfigWrapper {
	return appConfigWrapper{
		ProviderConfig: &minderv1.ProviderConfig{
			AutoRegistration: &minderv1.AutoRegistration{
				Entities: nil,
			},
		},
		GitHubApp: &minderv1.GitHubAppProviderConfig{
			Endpoint: proto.String("https://api.github.com/"),
		},
	}
}

// parseV1AppConfig parses the raw config into a GitHubAppProviderConfig struct
func parseV1AppConfig(rawCfg json.RawMessage) (
	*minderv1.ProviderConfig,
	*minderv1.GitHubAppProviderConfig,
	error,
) {
	var w appConfigWrapper

	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	// we used to have a required key on the gh app config, so we need to check both
	// until we migrate and can remove the old key
	ghAppConfig := w.GitHubApp
	if ghAppConfig == nil {
		ghAppConfig = w.GitHubAppOldKey
	}

	return w.ProviderConfig, ghAppConfig, nil
}

// ParseAndMergeV1AppConfig parses the raw config into a GitHubAppProviderConfig struct
func ParseAndMergeV1AppConfig(rawCfg json.RawMessage) (
	*minderv1.ProviderConfig,
	*minderv1.GitHubAppProviderConfig,
	error,
) {
	mergedConfig := getDefaultAppConfig()

	overrideProviderConfig, overrideConfig, err := parseV1AppConfig(rawCfg)
	if err != nil {
		return nil, nil, err
	}
	mw := appConfigWrapper{
		ProviderConfig: overrideProviderConfig,
		GitHubApp:      overrideConfig,
	}

	err = mergo.Map(&mergedConfig, &mw, mergo.WithOverride)
	if err != nil {
		return nil, nil, fmt.Errorf("error merging provider config: %w", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(mergedConfig); err != nil {
		return nil, nil, fmt.Errorf("error validating v1 provider config: %w", err)
	}

	// Validate the config according to the protobuf validation rules.
	if err := mergedConfig.GitHubApp.Validate(); err != nil {
		return nil, nil, fmt.Errorf("error validating GitHubApp v1 provider config: %w", err)
	}

	if mergedConfig.ProviderConfig != nil {
		if err := mergedConfig.ProviderConfig.Validate(); err != nil {
			return nil, nil, fmt.Errorf("error validating provider config: %w", err)
		}
	}

	// we used to have a required key on the gh app config, so we need to check both
	// until we migrate and can remove the old key
	ghAppConfig := mergedConfig.GitHubApp
	if ghAppConfig == nil {
		ghAppConfig = mergedConfig.GitHubAppOldKey
	}
	if ghAppConfig == nil {
		return nil, nil, fmt.Errorf("no GitHub App config found")
	}

	return mergedConfig.ProviderConfig, ghAppConfig, nil
}

// embedding the struct to expose its JSON tags
type appConfigWrapperWrite struct {
	*minderv1.ProviderConfig
	//nolint:lll
	GitHubApp *minderv1.GitHubAppProviderConfig `json:"github_app,omitempty" yaml:"github_app" mapstructure:"github_app" validate:"required"`
}

// MarshalV1AppConfig unmarshalls and then marshalls back to get rid of unknown keys before storing
func MarshalV1AppConfig(rawCfg json.RawMessage) (json.RawMessage, error) {
	// unmarsall the raw config to get the correct key and strip unknown keys
	providerCfg, appCfg, err := parseV1AppConfig(rawCfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing provider config: %w", err)
	}

	err = providerCfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating provider config: %w", err)
	}

	// marshall back because that's what we are storing
	w := appConfigWrapperWrite{
		ProviderConfig: providerCfg,
		GitHubApp:      appCfg,
	}

	return json.Marshal(w)
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

// GetMinderUserId returns the user id for the GitHub App user
func (g *GitHubAppDelegate) GetMinderUserId(ctx context.Context) (int64, error) {
	// Try to get this user ID from the GitHub API
	//nolint:errcheck // this will never error
	appUserName, _ := g.GetName(ctx)
	user, _, err := g.client.Users.Get(ctx, appUserName)
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
