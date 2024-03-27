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
	"fmt"
	"github.com/stacklok/minder/internal/db"
	gitclient "github.com/stacklok/minder/internal/providers/git"
	githubapp "github.com/stacklok/minder/internal/providers/github/app"
	ghclient "github.com/stacklok/minder/internal/providers/github/oauth"
	httpclient "github.com/stacklok/minder/internal/providers/http"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
	"slices"
)

type ProviderFactory interface {
	GetGit() (provinfv1.Git, error)
	GetHTTP() (provinfv1.REST, error)
	GetGitHub() (provinfv1.GitHub, error)
	GetRepoLister() (provinfv1.RepoLister, error)
}

type providerFactory struct {
}

// GetGit returns a git client for the provider.
func (p *providerFactory) GetGit() (provinfv1.Git, error) {
	if !p.Implements(db.ProviderTypeGit) {
		return nil, fmt.Errorf("provider does not implement git")
	}

	if p.Implements(db.ProviderTypeGithub) {
		return p.GetGitHub()
	}

	gitCredential, ok := pb.credential.(provinfv1.GitCredential)
	if !ok {
		return nil, ErrInvalidCredential
	}

	return gitclient.NewGit(gitCredential), nil
}

// GetHTTP returns a github client for the provider.
func (p *providerFactory) GetHTTP() (provinfv1.REST, error) {
	if !pb.Implements(db.ProviderTypeRest) {
		return nil, fmt.Errorf("provider does not implement rest")
	}

	// We can re-use the GitHub provider in case it also implements GitHub.
	// The client gives us the ability to handle rate limiting and other
	// things.
	if pb.Implements(db.ProviderTypeGithub) {
		return pb.GetGitHub()
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	// TODO: Parsing will change based on version
	cfg, err := httpclient.ParseV1Config(pb.p.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing http config: %w", err)
	}

	restCredential, ok := pb.credential.(provinfv1.RestCredential)
	if !ok {
		return nil, ErrInvalidCredential
	}

	return httpclient.NewREST(cfg, pb.metrics, restCredential)
}

// GetGitHub returns a github client for the provider.
func (p *providerFactory) GetGitHub() (provinfv1.GitHub, error) {
	if !pb.Implements(db.ProviderTypeGithub) {
		return nil, fmt.Errorf("provider does not implement github")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	gitHubCredential, ok := pb.credential.(provinfv1.GitHubCredential)
	if !ok {
		return nil, ErrInvalidCredential
	}

	if pb.restClientCache != nil {
		client, ok := pb.restClientCache.Get(pb.ownerFilter.String, gitHubCredential.GetCacheKey(), db.ProviderTypeGithub)
		if ok {
			return client.(provinfv1.GitHub), nil
		}
	}

	// TODO: use provider class once it's available
	if pb.p.Name == ghclient.Github {
		// TODO: Parsing will change based on version
		cfg, err := ghclient.ParseV1Config(pb.p.Definition)
		if err != nil {
			return nil, fmt.Errorf("error parsing github config: %w", err)
		}

		cli, err := ghclient.NewRestClient(cfg, pb.metrics, pb.restClientCache, gitHubCredential, pb.ownerFilter.String)
		if err != nil {
			return nil, fmt.Errorf("error creating github client: %w", err)
		}
		return cli, nil
	}

	cfg, err := githubapp.ParseV1Config(pb.p.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	cli, err := githubapp.NewGitHubAppProvider(cfg, pb.cfg.GitHubApp, pb.metrics, pb.restClientCache, gitHubCredential)
	if err != nil {
		return nil, fmt.Errorf("error creating github app client: %w", err)
	}
	return cli, nil
}

// GetRepoLister returns a repo lister for the provider.
func (p *providerFactory) GetRepoLister() (provinfv1.RepoLister, error) {
	if !pb.Implements(db.ProviderTypeRepoLister) {
		return nil, fmt.Errorf("provider does not implement repo lister")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	if pb.Implements(db.ProviderTypeGithub) {
		return pb.GetGitHub()
	}

	// TODO: We'll need to add support for other providers here
	return nil, fmt.Errorf("provider does not implement repo lister")
}

// Implements returns true if the provider implements the given type.
func (p *providerFactory) Implements(impl db.ProviderType) bool {
	return slices.Contains(p.p.Implements, impl)
}
