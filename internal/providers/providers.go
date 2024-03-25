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
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/dockerhub"
	gitclient "github.com/stacklok/minder/internal/providers/git"
	githubapp "github.com/stacklok/minder/internal/providers/github/app"
	"github.com/stacklok/minder/internal/providers/github/ghcr"
	ghclient "github.com/stacklok/minder/internal/providers/github/oauth"
	httpclient "github.com/stacklok/minder/internal/providers/http"
	"github.com/stacklok/minder/internal/providers/oci"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// ErrInvalidCredential is returned when the credential is not of the required type
var ErrInvalidCredential = errors.New("invalid credential type")

// GetProviderBuilder is a utility function which allows for the creation of
// a provider factory.
func GetProviderBuilder(
	ctx context.Context,
	prov db.Provider,
	store db.Store,
	crypteng crypto.Engine,
	provCfg *serverconfig.ProviderConfig,
	opts ...ProviderBuilderOption,
) (*ProviderBuilder, error) {
	credential, err := getCredentialForProvider(ctx, prov, crypteng, store, provCfg)
	if err != nil {
		return nil, fmt.Errorf("error getting credential: %w", err)
	}

	ownerFilter, err := getOwnerFilterForProvider(ctx, prov, store)
	if err != nil {
		return nil, fmt.Errorf("error getting owner filter: %w", err)
	}

	return NewProviderBuilder(&prov, ownerFilter, credential, provCfg, opts...), nil
}

// ProviderBuilder is a utility struct which allows for the creation of
// provider clients.
type ProviderBuilder struct {
	p               *db.Provider
	ownerFilter     sql.NullString // NOTE: we don't seem to actually use the null-ness anywhere.
	restClientCache ratecache.RestClientCache
	credential      provinfv1.Credential
	metrics         telemetry.ProviderMetrics
	cfg             *serverconfig.ProviderConfig
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
	ownerFilter sql.NullString,
	credential provinfv1.Credential,
	cfg *serverconfig.ProviderConfig,
	opts ...ProviderBuilderOption,
) *ProviderBuilder {
	pb := &ProviderBuilder{
		p:           p,
		cfg:         cfg,
		ownerFilter: ownerFilter,
		credential:  credential,
		metrics:     telemetry.NewNoopMetrics(),
	}

	for _, opt := range opts {
		opt(pb)
	}

	return pb
}

// Implements returns true if the provider implements the given type.
func (pb *ProviderBuilder) Implements(impl db.ProviderType) bool {
	return slices.Contains(pb.p.Implements, impl)
}

// GetName returns the name of the provider instance as defined in the
// database.
func (pb *ProviderBuilder) GetName() string {
	return pb.p.Name
}

// GetGit returns a git client for the provider.
func (pb *ProviderBuilder) GetGit() (provinfv1.Git, error) {
	if !pb.Implements(db.ProviderTypeGit) {
		return nil, fmt.Errorf("provider does not implement git")
	}

	if pb.Implements(db.ProviderTypeGithub) {
		return pb.GetGitHub()
	}

	gitCredential, ok := pb.credential.(provinfv1.GitCredential)
	if !ok {
		return nil, ErrInvalidCredential
	}

	return gitclient.NewGit(gitCredential), nil
}

// GetHTTP returns a github client for the provider.
func (pb *ProviderBuilder) GetHTTP() (provinfv1.REST, error) {
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
func (pb *ProviderBuilder) GetGitHub() (provinfv1.GitHub, error) {
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
func (pb *ProviderBuilder) GetRepoLister() (provinfv1.RepoLister, error) {
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

// GetGHCR returns a GHCR client for the provider.
func (pb *ProviderBuilder) GetGHCR() (provinfv1.ImageLister, error) {
	if !pb.Implements(db.ProviderTypeImageLister) {
		return nil, fmt.Errorf("provider does not implement ghcr")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	oauth2cred, ok := pb.credential.(provinfv1.OAuth2TokenCredential)
	if !ok {
		return nil, fmt.Errorf("credential is not an oauth2 token credential")
	}

	// TODO: Use config and not hardcode namespace
	return ghcr.New(oauth2cred, "jaormx"), nil
}

// GetDockerHub returns a DockerHub client for the provider.
func (pb *ProviderBuilder) GetDockerHub() (provinfv1.ImageLister, error) {
	if !pb.Implements(db.ProviderTypeImageLister) {
		return nil, fmt.Errorf("provider does not implement dockerhub")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	var cred provinfv1.Credential
	var ok bool
	cred, ok = pb.credential.(provinfv1.OAuth2TokenCredential)
	if !ok {
		if cred, ok = pb.credential.(*credentials.EmptyCredential); !ok {
			return nil, fmt.Errorf("credential is not an oauth2 token credential nor empty credential")
		}
	}

	return dockerhub.New(cred, "devopsfaith")
}

// GetImageLister returns an image lister for the provider.
func (pb *ProviderBuilder) GetImageLister() (provinfv1.ImageLister, error) {
	if pb.Implements(db.ProviderTypeDockerhub) {
		return pb.GetDockerHub()
	}

	if pb.Implements(db.ProviderTypeGhcr) {
		return pb.GetGHCR()
	}

	return nil, fmt.Errorf("provider does not implement image lister")
}

// GetOCI returns an OCI client for the provider.
func (pb *ProviderBuilder) GetOCI() (provinfv1.OCI, error) {
	if !pb.Implements(db.ProviderTypeOci) {
		return nil, fmt.Errorf("provider does not implement oci")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	return oci.New(pb.credential, "docker.io/devopsfaith"), nil
}

// DBToPBType converts a database provider type to a protobuf provider type.
func DBToPBType(t db.ProviderType) (minderv1.ProviderType, bool) {
	switch t {
	case db.ProviderTypeGit:
		return minderv1.ProviderType_PROVIDER_TYPE_GIT, true
	case db.ProviderTypeGithub:
		return minderv1.ProviderType_PROVIDER_TYPE_GITHUB, true
	case db.ProviderTypeRest:
		return minderv1.ProviderType_PROVIDER_TYPE_REST, true
	case db.ProviderTypeRepoLister:
		return minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER, true
	case db.ProviderTypeOci:
		return minderv1.ProviderType_PROVIDER_TYPE_OCI, true
	case db.ProviderTypeImageLister:
		return minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER, true
	case db.ProviderTypeGhcr:
		return minderv1.ProviderType_PROVIDER_TYPE_GHCR, true
	case db.ProviderTypeDockerhub:
		return minderv1.ProviderType_PROVIDER_TYPE_DOCKERHUB, true
	default:
		return minderv1.ProviderType_PROVIDER_TYPE_UNSPECIFIED, false
	}
}

// DBToPBAuthFlow converts a database authorization flow to a protobuf authorization flow.
func DBToPBAuthFlow(t db.AuthorizationFlow) (minderv1.AuthorizationFlow, bool) {
	switch t {
	case db.AuthorizationFlowNone:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_NONE, true
	case db.AuthorizationFlowUserInput:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT, true
	case db.AuthorizationFlowOauth2AuthorizationCodeFlow:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW, true
	case db.AuthorizationFlowGithubAppFlow:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_GITHUB_APP_FLOW, true
	default:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_UNSPECIFIED, false
	}
}

// GetCredentialStateForProvider returns the credential state for the given provider.
func GetCredentialStateForProvider(
	ctx context.Context,
	prov db.Provider,
	s db.Store,
	cryptoEngine crypto.Engine,
	provCfg *serverconfig.ProviderConfig,
) string {
	var credState string
	// if the provider doesn't support any auth flow
	// credentials state is not applicable
	if slices.Equal(prov.AuthFlows, []db.AuthorizationFlow{db.AuthorizationFlowNone}) {
		credState = provinfv1.CredentialStateNotApplicable
	} else {
		credState = provinfv1.CredentialStateUnset
		cred, err := getCredentialForProvider(ctx, prov, cryptoEngine, s, provCfg)
		if err != nil {
			// This is non-fatal
			zerolog.Ctx(ctx).Error().Err(err).Str("provider", prov.Name).Msg("error getting credential")
		} else {
			// check if the credential is EmptyCredential
			// if it is, then the state is not applicable
			if _, ok := cred.(*credentials.EmptyCredential); ok {
				credState = provinfv1.CredentialStateUnset
			} else {
				credState = provinfv1.CredentialStateSet
			}
		}
	}

	return credState
}

func getCredentialForProvider(
	ctx context.Context,
	prov db.Provider,
	crypteng crypto.Engine,
	store db.Store,
	cfg *serverconfig.ProviderConfig,
) (provinfv1.Credential, error) {
	encToken, err := store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: prov.Name, ProjectID: prov.ProjectID})
	if errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Debug().Msg("no access token found for provider")

		// If we don't have an access token, check if we have an installation ID
		return getInstallationTokenCredential(ctx, prov, store, cfg)
	} else if err != nil {
		return nil, fmt.Errorf("error getting credential: %w", err)
	}

	decryptedToken, err := crypteng.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msg("access token found for provider")
	return credentials.NewGitHubTokenCredential(decryptedToken.AccessToken), nil
}

// getInstallationTokenCredential returns a GitHub installation token credential if the provider has an installation ID
func getInstallationTokenCredential(
	ctx context.Context,
	prov db.Provider,
	store db.Store,
	provCfg *serverconfig.ProviderConfig,
) (provinfv1.Credential, error) {
	installation, err := store.GetInstallationIDByProviderID(ctx, uuid.NullUUID{
		UUID:  prov.ID,
		Valid: true,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return credentials.NewEmptyCredential(), nil
	} else if err != nil {
		return nil, fmt.Errorf("error getting installation ID: %w", err)
	}
	cfg, err := githubapp.ParseV1Config(prov.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	privateKeyBytes, err := provCfg.GitHubApp.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	return credentials.NewGitHubInstallationTokenCredential(ctx, provCfg.GitHubApp.AppID, privateKey, cfg.Endpoint,
		installation.AppInstallationID), nil
}

func getOwnerFilterForProvider(ctx context.Context, prov db.Provider, store db.Store) (sql.NullString, error) {
	encToken, err := store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: prov.Name, ProjectID: prov.ProjectID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.NullString{}, nil
		}
		return sql.NullString{}, fmt.Errorf("error getting access token: %w", err)
	}

	return encToken.OwnerFilter, nil
}
