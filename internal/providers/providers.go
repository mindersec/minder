// Copyright 2023 Stacklok, Inc.
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

// Package providers contains general utilities for interacting with
// providers.
package providers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"

	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	gitclient "github.com/stacklok/minder/internal/providers/git"
	ghclient "github.com/stacklok/minder/internal/providers/github"
	httpclient "github.com/stacklok/minder/internal/providers/http"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GetProviderBuilder is a utility function which allows for the creation of
// a provider factory.
func GetProviderBuilder(
	ctx context.Context,
	prov db.Provider,
	projectID uuid.UUID,
	store db.Store,
	crypteng *crypto.Engine,
	opts ...ProviderBuilderOption,
) (*ProviderBuilder, error) {
	encToken, err := store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: prov.Name, ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}

	decryptedToken, err := crypteng.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}

	return NewProviderBuilder(&prov, encToken, decryptedToken.AccessToken, opts...), nil
}

// ProviderBuilder is a utility struct which allows for the creation of
// provider clients.
type ProviderBuilder struct {
	p               *db.Provider
	tokenInf        db.ProviderAccessToken
	restClientCache ratecache.RestClientCache
	tok             string
	metrics         telemetry.ProviderMetrics
}

// ProviderBuilderOption is a function which can be used to set options on the ProviderBuilder.
type ProviderBuilderOption func(*ProviderBuilder)

// WithProviderMetrics sets the metrics for the ProviderBuilder
func WithProviderMetrics(metrics telemetry.ProviderMetrics) ProviderBuilderOption {
	return func(pb *ProviderBuilder) {
		pb.metrics = metrics
	}
}

// WithRestClientCache sets the rest client cache for the ProviderBuilder
func WithRestClientCache(cache ratecache.RestClientCache) ProviderBuilderOption {
	return func(pb *ProviderBuilder) {
		pb.restClientCache = cache
	}
}

// NewProviderBuilder creates a new provider builder.
func NewProviderBuilder(
	p *db.Provider,
	tokenInf db.ProviderAccessToken,
	tok string,
	opts ...ProviderBuilderOption,
) *ProviderBuilder {
	pb := &ProviderBuilder{
		p:        p,
		tokenInf: tokenInf,
		tok:      tok,
		metrics:  telemetry.NewNoopMetrics(),
	}

	for _, opt := range opts {
		opt(pb)
	}

	return pb
}

// Implements returns true if the provider implements the given type.
func (pb *ProviderBuilder) Implements(impl db.ProviderTrait) bool {
	return slices.Contains(pb.p.Implements, impl)
}

// GetName returns the name of the provider instance as defined in the
// database.
func (pb *ProviderBuilder) GetName() string {
	return pb.p.Name
}

// GetToken returns the token for the provider.
func (pb *ProviderBuilder) GetToken() string {
	return pb.tok
}

// GetGit returns a git client for the provider.
func (pb *ProviderBuilder) GetGit() (*gitclient.Git, error) {
	if !pb.Implements(db.ProviderTraitGit) {
		return nil, fmt.Errorf("provider does not implement git")
	}

	return gitclient.NewGit(pb.tok), nil
}

// GetHTTP returns a github client for the provider.
func (pb *ProviderBuilder) GetHTTP() (provinfv1.REST, error) {
	if !pb.Implements(db.ProviderTraitRest) {
		return nil, fmt.Errorf("provider does not implement rest")
	}

	// We can re-use the GitHub provider in case it also implements GitHub.
	// The client gives us the ability to handle rate limiting and other
	// things.
	if pb.Implements(db.ProviderTraitGithub) {
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

	return httpclient.NewREST(cfg, pb.metrics, pb.tok)
}

// GetGitHub returns a github client for the provider.
func (pb *ProviderBuilder) GetGitHub() (provinfv1.GitHub, error) {
	if !pb.Implements(db.ProviderTraitGithub) {
		return nil, fmt.Errorf("provider does not implement github")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	if pb.restClientCache != nil {
		client, ok := pb.restClientCache.Get(pb.tokenInf.OwnerFilter.String, pb.GetToken(), db.ProviderTraitGithub)
		if ok {
			return client.(provinfv1.GitHub), nil
		}
	}

	// TODO: Parsing will change based on version
	cfg, err := ghclient.ParseV1Config(pb.p.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github config: %w", err)
	}

	cli, err := ghclient.NewRestClient(cfg, pb.metrics, pb.restClientCache, pb.GetToken(), pb.tokenInf.OwnerFilter.String)
	if err != nil {
		return nil, fmt.Errorf("error creating github client: %w", err)
	}

	return cli, nil
}

// GetRepoLister returns a repo lister for the provider.
func (pb *ProviderBuilder) GetRepoLister() (provinfv1.RepoLister, error) {
	if !pb.Implements(db.ProviderTraitRepoLister) {
		return nil, fmt.Errorf("provider does not implement repo lister")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	if pb.Implements(db.ProviderTraitGithub) {
		return pb.GetGitHub()
	}

	// TODO: We'll need to add support for other providers here
	return nil, fmt.Errorf("provider does not implement repo lister")
}

// DBToPBType converts a database provider type to a protobuf provider type.
func DBToPBType(t db.ProviderTrait) (minderv1.ProviderType, bool) {
	switch t {
	case db.ProviderTraitGit:
		return minderv1.ProviderType_PROVIDER_TYPE_GIT, true
	case db.ProviderTraitGithub:
		return minderv1.ProviderType_PROVIDER_TYPE_GITHUB, true
	case db.ProviderTraitRest:
		return minderv1.ProviderType_PROVIDER_TYPE_REST, true
	case db.ProviderTraitRepoLister:
		return minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER, true
	case db.ProviderTraitOci:
		return minderv1.ProviderType_PROVIDER_TYPE_OCI, true
	default:
		return minderv1.ProviderType_PROVIDER_TYPE_UNSPECIFIED, false
	}
}
