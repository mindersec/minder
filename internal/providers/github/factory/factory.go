// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package factory contains the GitHubProviderFactory
package factory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	gogithub "github.com/google/go-github/v61/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/factory"
	githubapp "github.com/stacklok/minder/internal/providers/github/app"
	ghclient "github.com/stacklok/minder/internal/providers/github/oauth"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// NewGitHubProviderClassFactory creates an instance of ProviderClassFactory
// for creating GitHub-specific provider instances.
func NewGitHubProviderClassFactory(
	restClientCache ratecache.RestClientCache,
	metrics telemetry.HttpClientMetrics,
	config *server.GitHubAppConfig,
	fallbackTokenClient *gogithub.Client,
	crypteng crypto.Engine,
	store db.Store,
) factory.ProviderClassFactory {
	return &githubProviderFactory{
		restClientCache:     restClientCache,
		metrics:             metrics,
		config:              config,
		fallbackTokenClient: fallbackTokenClient,
		crypteng:            crypteng,
		store:               store,
	}
}

type githubProviderFactory struct {
	restClientCache     ratecache.RestClientCache
	metrics             telemetry.HttpClientMetrics
	config              *server.GitHubAppConfig
	fallbackTokenClient *gogithub.Client
	crypteng            crypto.Engine
	store               db.Store
}

var (
	supportedClasses = []db.ProviderClass{
		db.ProviderClassGithubApp,
		db.ProviderClassGithub,
	}
)

func (_ *githubProviderFactory) GetSupportedClasses() []db.ProviderClass {
	return supportedClasses
}

func (g *githubProviderFactory) Build(ctx context.Context, config *db.Provider) (v1.Provider, error) {
	class := config.Class
	// This should be validated by the caller, but let's check anyway
	if !slices.Contains(supportedClasses, class) {
		return nil, fmt.Errorf("provider does not implement github")
	}

	if config.Version != v1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	creds, err := g.getProviderCredentials(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch credentials")
	}

	if g.restClientCache != nil {
		client, ok := g.restClientCache.Get(creds.ownerFilter.String, creds.credential.GetCacheKey(), db.ProviderTypeGithub)
		if ok {
			return client.(v1.GitHub), nil
		}
	}

	// previously this was done by checking the name, I think this is safer
	if class == db.ProviderClassGithub {
		// TODO: Parsing will change based on version
		cfg, err := ghclient.ParseV1Config(config.Definition)
		if err != nil {
			return nil, fmt.Errorf("error parsing github config: %w", err)
		}

		cli, err := ghclient.NewRestClient(cfg, g.metrics, g.restClientCache, creds.credential, creds.ownerFilter.String)
		if err != nil {
			return nil, fmt.Errorf("error creating github client: %w", err)
		}
		return cli, nil
	}

	cfg, err := githubapp.ParseV1Config(config.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	cli, err := githubapp.NewGitHubAppProvider(cfg, g.config, g.metrics, g.restClientCache, creds.credential,
		g.fallbackTokenClient, creds.isOrg)
	if err != nil {
		return nil, fmt.Errorf("error creating github app client: %w", err)
	}
	return cli, nil
}

func (g *githubProviderFactory) createProviderWithAccessToken(
	ctx context.Context,
	encToken db.ProviderAccessToken,
) (*credentialDetails, error) {
	decryptedToken, err := g.crypteng.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msg("access token found for provider")

	credential := credentials.NewGitHubTokenCredential(decryptedToken.AccessToken)
	ownerFilter := encToken.OwnerFilter
	isOrg := ownerFilter != sql.NullString{} && ownerFilter.String != ""

	return &credentialDetails{
		credential:  credential,
		ownerFilter: ownerFilter,
		isOrg:       isOrg,
	}, nil
}

func (g *githubProviderFactory) getProviderCredentials(
	ctx context.Context,
	prov *db.Provider,
) (*credentialDetails, error) {
	encToken, err := g.store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: prov.Name, ProjectID: prov.ProjectID})
	if errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Debug().Msg("no access token found for provider")

		// If we don't have an access token, check if we have an installation ID
		return g.createProviderWithInstallationToken(ctx, prov)
	} else if err != nil {
		return nil, fmt.Errorf("error getting credential: %w", err)
	}

	return g.createProviderWithAccessToken(ctx, encToken)
}

func (g *githubProviderFactory) createProviderWithInstallationToken(
	ctx context.Context,
	prov *db.Provider,
) (*credentialDetails, error) {
	installation, err := g.store.GetInstallationIDByProviderID(ctx, uuid.NullUUID{
		UUID:  prov.ID,
		Valid: true,
	})

	ownerFilter := sql.NullString{}
	if errors.Is(err, sql.ErrNoRows) {
		// If the provider doesn't have a known credential set the credential to empty
		return &credentialDetails{
			credential:  &credentials.GitHubInstallationTokenCredential{},
			ownerFilter: ownerFilter,
			isOrg:       false,
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("error getting installation ID: %w", err)
	}

	cfg, err := githubapp.ParseV1Config(prov.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	privateKey, err := g.config.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %w", err)
	}

	credential := credentials.NewGitHubInstallationTokenCredential(ctx, g.config.AppID, privateKey, cfg.Endpoint,
		installation.AppInstallationID)

	zerolog.Ctx(ctx).
		Debug().
		Str("github-app-name", g.config.AppName).
		Int64("github-app-id", g.config.AppID).
		Int64("github-app-installation-id", installation.AppInstallationID).
		Msg("created provider with installation token")

	return &credentialDetails{
		credential:  credential,
		ownerFilter: ownerFilter,
		isOrg:       installation.IsOrg,
	}, nil
}

type credentialDetails struct {
	credential  v1.GitHubCredential
	ownerFilter sql.NullString
	isOrg       bool
}
