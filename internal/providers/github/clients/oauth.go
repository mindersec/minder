// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"encoding/json"
	"fmt"

	"dario.cat/mergo"
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

// Github is the string that represents the GitHubOAuth provider
const Github = "github"

// OAuthImplements is the list of provider types that the GitHubOAuth provider implements
var OAuthImplements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// OAuthAuthorizationFlows is the list of authorization flows that the GitHubOAuth provider supports
var OAuthAuthorizationFlows = []db.AuthorizationFlow{
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

// NewOAuthDelegate creates a GitHubOAuthDelegate from a GitHub client
// This exists as a separate function to allow the provider creation code
// to use its methods without instantiating a full provider.
func NewOAuthDelegate(
	client *gogithub.Client,
	credential provifv1.GitHubCredential,
	owner string,
) *GitHubOAuthDelegate {
	return &GitHubOAuthDelegate{
		client:     client,
		credential: credential,
		owner:      owner,
	}
}

// NewRestClient creates a new GitHub REST API client
// BaseURL defaults to the public GitHub API, if needing to use a customer domain
// endpoint (as is the case with GitHub Enterprise), set the Endpoint field in
// the GitHubConfig struct
func NewRestClient(
	cfg *minderv1.GitHubProviderConfig,
	providerCfg *server.ProviderConfig,
	whcfg *server.WebhookConfig,
	restClientCache ratecache.RestClientCache,
	credential provifv1.GitHubCredential,
	ghClientFactory GitHubClientFactory,
	propertyFetchers properties.GhPropertyFetcherFactory,
	owner string,
) (*github.GitHub, error) {
	ghClient, delegate, err := ghClientFactory.BuildOAuthClient(cfg.GetEndpoint(), credential, owner)
	if err != nil {
		return nil, err
	}

	return github.NewGitHub(
		ghClient,
		ghClient, // use the same client for listing packages and all other operations
		restClientCache,
		delegate,
		providerCfg,
		whcfg,
		propertyFetchers,
	), nil
}

type oauthConfigWrapper struct {
	GitHub *minderv1.GitHubProviderConfig `json:"github,omitempty" yaml:"github" mapstructure:"github" validate:"required"`
}

func getDefaultOAuthConfig() oauthConfigWrapper {
	return oauthConfigWrapper{
		GitHub: &minderv1.GitHubProviderConfig{
			Endpoint: proto.String("https://api.github.com/"),
		},
	}
}

// ParseAndMergeV1OAuthConfig parses the raw config into a GitHubConfig struct
func ParseAndMergeV1OAuthConfig(rawCfg json.RawMessage) (*minderv1.GitHubProviderConfig, error) {
	mergedCfg := getDefaultOAuthConfig()

	var w oauthConfigWrapper
	if err := json.Unmarshal(rawCfg, &w); err != nil {
		return nil, fmt.Errorf("error parsing GitHubOAuth v1 provider user config: %w", err)
	}

	if err := mergo.Map(&mergedCfg, w, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("error merging GitHubOAuth v1 provider config: %w", err)
	}

	// Validate the config according to the protobuf validation rules.
	if err := mergedCfg.GitHub.Validate(); err != nil {
		return nil, fmt.Errorf("error validating GitHubOAuth v1 provider config: %w", err)
	}

	return mergedCfg.GitHub, nil
}

// MarshalV1OAuthConfig marshals the GitHubConfig struct into a raw config
func MarshalV1OAuthConfig(rawCfg json.RawMessage) (json.RawMessage, error) {
	var w oauthConfigWrapper
	if err := json.Unmarshal(rawCfg, &w); err != nil {
		return nil, err
	}

	err := w.GitHub.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating provider config: %w", err)
	}

	return json.Marshal(w)
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

// GetMinderUserId returns the user id for the authenticated user
func (o *GitHubOAuthDelegate) GetMinderUserId(ctx context.Context) (int64, error) {
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
