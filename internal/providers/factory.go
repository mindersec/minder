// Copyright 2024 Stacklok, Inc.
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

package providers

import (
	"context"
	"fmt"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	gitclient "github.com/stacklok/minder/internal/providers/git"
	githubapp "github.com/stacklok/minder/internal/providers/github/app"
	ghclient "github.com/stacklok/minder/internal/providers/github/oauth"
	httpclient "github.com/stacklok/minder/internal/providers/http"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// TraitInstantiator is responsible for creating instances of the Profile
// interfaces defined in `github.com/stacklok/minder/pkg/providers/v1` from
// the providers stored in the database.
type TraitInstantiator interface {
	GetGit(ctx context.Context, provider *db.Provider) (provinfv1.Git, error)
	GetHTTP(ctx context.Context, provider *db.Provider) (provinfv1.REST, error)
	GetGitHub(ctx context.Context, provider *db.Provider, options *GitHubOptions) (provinfv1.GitHub, error)
	GetRepoLister(ctx context.Context, provider *db.Provider) (provinfv1.RepoLister, error)
}

// GitHubOptions contains options used when creating a GitHub instance
// If omitted, these will be looked up in the database
type GitHubOptions struct {
	Credential  provinfv1.GitHubCredential
	OwnerFilter *string
}

type traitInstantiator struct {
	restClientCache ratecache.RestClientCache
	metrics         telemetry.ProviderMetrics
	cfg             *serverconfig.ProviderConfig
	store           db.Store
	crypteng        crypto.Engine
}

// NewTraitInstantiator creates a new instance of TraitInstantiator
func NewTraitInstantiator(
	restClientCache ratecache.RestClientCache,
	metrics telemetry.ProviderMetrics,
	cfg *serverconfig.ProviderConfig,
	store db.Store,
	crypteng crypto.Engine,
) TraitInstantiator {
	return &traitInstantiator{
		restClientCache: restClientCache,
		metrics:         metrics,
		cfg:             cfg,
		store:           store,
		crypteng:        crypteng,
	}
}

// GetGit returns a git client for the provider.
func (t *traitInstantiator) GetGit(ctx context.Context, provider *db.Provider) (provinfv1.Git, error) {
	if !provider.CanImplement(db.ProviderTypeGit) {
		return nil, fmt.Errorf("provider does not implement git")
	}

	if provider.CanImplement(db.ProviderTypeGithub) {
		return t.GetGitHub(ctx, provider, nil)
	}

	gitCredential, err := getCredential[provinfv1.GitCredential](ctx, t, provider, nil)
	if err != nil {
		return nil, err
	}

	return gitclient.NewGit(gitCredential), nil
}

// GetHTTP returns a github client for the provider.
func (t *traitInstantiator) GetHTTP(ctx context.Context, provider *db.Provider) (provinfv1.REST, error) {
	if !provider.CanImplement(db.ProviderTypeRest) {
		return nil, fmt.Errorf("provider does not implement rest")
	}

	// We can re-use the GitHub provider in case it also implements GitHub.
	// The client gives us the ability to handle rate limiting and other
	// things.
	if provider.CanImplement(db.ProviderTypeGithub) {
		return t.GetGitHub(ctx, provider, nil)
	}

	if provider.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	restCredential, err := getCredential[provinfv1.RestCredential](ctx, t, provider, nil)
	if err != nil {
		return nil, err
	}

	// TODO: Parsing will change based on version
	cfg, err := httpclient.ParseV1Config(provider.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing http config: %w", err)
	}

	return httpclient.NewREST(cfg, t.metrics, restCredential)
}

// GetGitHub returns a github client for the provider.
func (t *traitInstantiator) GetGitHub(
	ctx context.Context,
	provider *db.Provider,
	options *GitHubOptions,
) (provinfv1.GitHub, error) {
	if !provider.CanImplement(db.ProviderTypeGithub) {
		return nil, fmt.Errorf("provider does not implement github")
	}

	if provider.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	credential, err := getCredential[provinfv1.GitHubCredential](ctx, t, provider, options)
	if err != nil {
		return nil, err
	}

	filter, err := t.getOwnerFilter(ctx, provider, options)
	if err != nil {
		return nil, err
	}

	if t.restClientCache != nil {
		client, ok := t.restClientCache.Get(filter, credential.GetCacheKey(), db.ProviderTypeGithub)
		if ok {
			return client.(provinfv1.GitHub), nil
		}
	}

	// TODO: use provider class once it's available
	if provider.Name == ghclient.Github {
		// TODO: Parsing will change based on version
		cfg, err := ghclient.ParseV1Config(provider.Definition)
		if err != nil {
			return nil, fmt.Errorf("error parsing github config: %w", err)
		}

		cli, err := ghclient.NewRestClient(cfg, t.metrics, t.restClientCache, credential, filter)
		if err != nil {
			return nil, fmt.Errorf("error creating github client: %w", err)
		}
		return cli, nil
	}

	cfg, err := githubapp.ParseV1Config(provider.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	cli, err := githubapp.NewGitHubAppProvider(cfg, t.cfg.GitHubApp, t.metrics, t.restClientCache, credential)
	if err != nil {
		return nil, fmt.Errorf("error creating github app client: %w", err)
	}
	return cli, nil
}

// GetRepoLister returns a repo lister for the provider.
func (t *traitInstantiator) GetRepoLister(ctx context.Context, provider *db.Provider) (provinfv1.RepoLister, error) {
	if !provider.CanImplement(db.ProviderTypeRepoLister) {
		return nil, fmt.Errorf("provider does not implement repo lister")
	}

	if provider.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	if provider.CanImplement(db.ProviderTypeGithub) {
		return t.GetGitHub(ctx, provider, nil)
	}

	// TODO: We'll need to add support for other providers here
	return nil, fmt.Errorf("provider does not implement repo lister")
}

func (t *traitInstantiator) getCredentialFromDB(
	ctx context.Context,
	provider *db.Provider,
) (provinfv1.Credential, error) {
	credential, err := getCredentialForProvider(ctx, *provider, t.crypteng, t.store, t.cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting credential: %w", err)
	}
	return credential, nil
}

// Because of the use of generics, this cannot be a method of traitInstantiator
func getCredential[T provinfv1.Credential](
	ctx context.Context,
	t *traitInstantiator,
	provider *db.Provider,
	options *GitHubOptions,
) (result T, err error) {
	var credential provinfv1.Credential
	if options == nil || options.Credential == nil {
		credential, err = t.getCredentialFromDB(ctx, provider)
		if err != nil {
			return
		}
	} else {
		credential = options.Credential
	}

	var ok bool
	result, ok = credential.(T)
	if !ok {
		return result, ErrInvalidCredential
	}
	return result, nil
}

func (t *traitInstantiator) getOwnerFilter(
	ctx context.Context,
	provider *db.Provider,
	options *GitHubOptions,
) (string, error) {
	if options != nil && options.OwnerFilter != nil {
		return *options.OwnerFilter, nil
	}
	ownerFilter, err := getOwnerFilterForProvider(ctx, *provider, t.store)
	if err != nil {
		return "", fmt.Errorf("error getting owner filter: %w", err)
	}
	return ownerFilter.String, nil
}
