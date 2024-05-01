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

// Package manager contains the GitHubProviderClassManager
package manager

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
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/credentials"
	githubapp "github.com/stacklok/minder/internal/providers/github/app"
	"github.com/stacklok/minder/internal/providers/github/clients"
	ghclient "github.com/stacklok/minder/internal/providers/github/oauth"
	"github.com/stacklok/minder/internal/providers/github/service"
	m "github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// NewGitHubProviderClassManager creates an instance of ProviderClassManager
// for creating GitHub-specific provider instances.
func NewGitHubProviderClassManager(
	restClientCache ratecache.RestClientCache,
	ghClientFactory clients.GitHubClientFactory,
	providerConfig *server.ProviderConfig,
	fallbackTokenClient *gogithub.Client,
	crypteng crypto.Engine,
	whManager webhooks.WebhookManager,
	store db.Store,
	ghService service.GitHubProviderService,
) m.ProviderClassManager {
	return &githubProviderManager{
		restClientCache:     restClientCache,
		ghClientFactory:     ghClientFactory,
		config:              providerConfig,
		fallbackTokenClient: fallbackTokenClient,
		crypteng:            crypteng,
		store:               store,
		webhooks:            whManager,
		ghService:           ghService,
	}
}

type githubProviderManager struct {
	restClientCache     ratecache.RestClientCache
	ghClientFactory     clients.GitHubClientFactory
	config              *server.ProviderConfig
	fallbackTokenClient *gogithub.Client
	crypteng            crypto.Engine
	webhooks            webhooks.WebhookManager
	store               db.Store
	ghService           service.GitHubProviderService
}

var (
	supportedClasses = []db.ProviderClass{
		db.ProviderClassGithubApp,
		db.ProviderClassGithub,
	}
)

func (_ *githubProviderManager) GetSupportedClasses() []db.ProviderClass {
	return supportedClasses
}

func (g *githubProviderManager) Build(ctx context.Context, config *db.Provider) (v1.Provider, error) {
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

	client, ok := g.restClientCache.Get(creds.ownerFilter.String, creds.credential.GetCacheKey(), db.ProviderTypeGithub)
	if ok {
		return client.(v1.GitHub), nil
	}

	// previously this was done by checking the name, I think this is safer
	if class == db.ProviderClassGithub {
		// TODO: Parsing will change based on version
		cfg, err := ghclient.ParseV1Config(config.Definition)
		if err != nil {
			return nil, fmt.Errorf("error parsing github config: %w", err)
		}

		cli, err := ghclient.NewRestClient(
			cfg,
			g.restClientCache,
			creds.credential,
			g.ghClientFactory,
			creds.ownerFilter.String,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating github client: %w", err)
		}
		return cli, nil
	}

	cfg, err := githubapp.ParseV1Config(config.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	cli, err := githubapp.NewGitHubAppProvider(
		cfg,
		g.config.GitHubApp,
		g.restClientCache,
		creds.credential,
		g.fallbackTokenClient,
		g.ghClientFactory,
		creds.isOrg,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating github app client: %w", err)
	}
	return cli, nil
}

func (g *githubProviderManager) Delete(ctx context.Context, config *db.Provider) error {
	state := providers.GetCredentialStateForProvider(ctx, *config, g.store, g.crypteng, g.config)
	if state == v1.CredentialStateSet {
		provider, err := g.Build(ctx, config)
		if err != nil {
			// errors from `Build` are good enough - no need to add extra context
			return err
		}

		client, err := v1.As[v1.GitHub](provider)
		if err != nil {
			// this should never happen
			return errors.New("unable to instantiate provider as GitHub client")
		}

		// to avoid a circular dependency between this struct and the repo
		// service, only delete the webhooks from the repos, and allow the
		// cascade delete to delete the repos in the DB
		// TODO: move webhook management behind a provider trait
		providerWebhooks, err := g.store.GetProviderWebhooks(ctx, config.ID)
		if err != nil {
			return fmt.Errorf("unable to retrieve list of webhooks: %w", err)
		}

		for _, webhook := range providerWebhooks {
			// SQL query guarantees that webhook ID is always non-null
			err = g.webhooks.DeleteWebhook(ctx, client, webhook.RepoOwner, webhook.RepoName, webhook.WebhookID.Int64)
			// Don't fail the deletion if the repositories cannot be deleted or webhook cannot be removed
			// The repositories will still be deleted by a cascade delete in the database
			zerolog.Ctx(ctx).Error().Err(err).
				Str("projectID", config.ProjectID.String()).
				Str("providerID", config.ID.String()).
				Msg("error deleting repositories")
		}
	}

	// clean up the app installation (if any)
	return g.ghService.DeleteInstallation(ctx, config.ID)
}

func (g *githubProviderManager) createProviderWithAccessToken(
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

func (g *githubProviderManager) getProviderCredentials(
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

func (g *githubProviderManager) createProviderWithInstallationToken(
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

	privateKey, err := g.config.GitHubApp.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %w", err)
	}

	credential := credentials.NewGitHubInstallationTokenCredential(ctx, g.config.GitHubApp.AppID, privateKey, cfg.Endpoint,
		installation.AppInstallationID)

	zerolog.Ctx(ctx).
		Debug().
		Str("github-app-name", g.config.GitHubApp.AppName).
		Int64("github-app-id", g.config.GitHubApp.AppID).
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
