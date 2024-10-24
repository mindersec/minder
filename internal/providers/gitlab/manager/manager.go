// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package manager contains the GitLabProviderClassManager
package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"github.com/sqlc-dev/pqtype"
	"golang.org/x/oauth2"

	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/gitlab"
	"github.com/mindersec/minder/pkg/config/server"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
)

// tokenExpirationThreshold is the time before the token expires that we should
// consider it expired and refresh it.
var tokenExpirationThreshold = -10 * time.Minute

type providerClassManager struct {
	store    db.Store
	crypteng crypto.Engine
	// gitlab provider config
	glpcfg        *server.GitLabConfig
	webhookURL    string
	parentContext context.Context
	pub           events.Publisher

	// secrets for the webhook. These are stored in the
	// structure to allow efficient fetching. Rotation
	// requires a process restart.
	currentWebhookSecret   string
	previousWebhookSecrets []string
}

// NewGitLabProviderClassManager creates a new provider class manager for the dockerhub provider
func NewGitLabProviderClassManager(
	ctx context.Context, crypteng crypto.Engine, store db.Store, pub events.Publisher,
	cfg *server.GitLabConfig, wgCfg server.WebhookConfig,
) (*providerClassManager, error) {
	webhookURLBase := wgCfg.ExternalWebhookURL
	if webhookURLBase == "" {
		return nil, errors.New("webhook URL is required")
	}

	if cfg == nil {
		return nil, errors.New("gitlab config is required")
	}

	webhookURL, err := url.JoinPath(webhookURLBase, url.PathEscape(string(db.ProviderClassGitlab)))
	if err != nil {
		return nil, fmt.Errorf("error joining webhook URL: %w", err)
	}

	whSecret, err := cfg.GetWebhookSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}

	previousSecrets, err := cfg.GetPreviousWebhookSecrets()
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("previous secrets not loaded")
	}

	return &providerClassManager{
		store:                  store,
		crypteng:               crypteng,
		pub:                    pub,
		glpcfg:                 cfg,
		webhookURL:             webhookURL,
		parentContext:          ctx,
		currentWebhookSecret:   whSecret,
		previousWebhookSecrets: previousSecrets,
	}, nil
}

// GetSupportedClasses implements the ProviderClassManager interface
func (_ *providerClassManager) GetSupportedClasses() []db.ProviderClass {
	return []db.ProviderClass{db.ProviderClassGitlab}
}

// Build implements the ProviderClassManager interface
func (g *providerClassManager) Build(ctx context.Context, config *db.Provider) (v1.Provider, error) {
	class := config.Class
	// This should be validated by the caller, but let's check anyway
	if !slices.Contains(g.GetSupportedClasses(), class) {
		return nil, fmt.Errorf("provider does not implement gitlab")
	}

	if config.Version != v1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	creds, err := g.getProviderCredentials(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch credentials: %w", err)
	}

	cfg, err := gitlab.ParseV1Config(config.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing gitlab config: %w", err)
	}

	cli, err := gitlab.New(creds, cfg, g.webhookURL, g.currentWebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("error creating gitlab client: %w", err)
	}
	return cli, nil
}

// Delete implements the ProviderClassManager interface
// TODO: Implement this
func (_ *providerClassManager) Delete(_ context.Context, _ *db.Provider) error {
	return nil
}

func (m *providerClassManager) getProviderCredentials(
	ctx context.Context,
	prov *db.Provider,
) (v1.GitLabCredential, error) {
	encToken, err := m.store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: prov.Name, ProjectID: prov.ProjectID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error getting credential: %w", err)
	}

	if !encToken.EncryptedAccessToken.Valid {
		return nil, fmt.Errorf("no secret found for provider %s", encToken.Provider)
	}

	encryptedData, err := crypto.DeserializeEncryptedData(encToken.EncryptedAccessToken.RawMessage)
	if err != nil {
		return nil, err
	}
	decryptedToken, err := m.crypteng.DecryptOAuthToken(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}

	if tokenNeedsRefresh(decryptedToken) {
		newtoken, err := m.refreshToken(ctx, decryptedToken.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("error refreshing token: %w", err)
		}

		if err := m.persistToken(ctx, prov, newtoken); err != nil {
			return nil, fmt.Errorf("error persisting refreshed token: %w", err)
		}

		zerolog.Ctx(ctx).Debug().
			Str("provider", prov.Name).
			Str("provider_class", string(prov.Class)).
			Str("project_id", prov.ProjectID.String()).
			Msg("refreshed token")

		decryptedToken = *newtoken
	}

	return credentials.NewGitLabTokenCredential(decryptedToken.AccessToken), nil
}

func (m *providerClassManager) MarshallConfig(
	_ context.Context, class db.ProviderClass, config json.RawMessage,
) (json.RawMessage, error) {
	if !slices.Contains(m.GetSupportedClasses(), class) {
		return nil, fmt.Errorf("provider does not implement %s", string(class))
	}

	return gitlab.MarshalV1Config(config)
}

func (m *providerClassManager) refreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	oauthcfg, err := m.NewOAuthConfig(db.ProviderClassGitlab, false)
	if err != nil {
		return nil, fmt.Errorf("error creating oauth config: %w", err)
	}

	newtoken, err := oauthcfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	}).Token()
	if err != nil {
		return nil, fmt.Errorf("error refreshing token: %w", err)
	}

	return newtoken, nil
}

func (m *providerClassManager) persistToken(
	ctx context.Context, prov *db.Provider, token *oauth2.Token,
) error {
	encryptedToken, err := m.crypteng.EncryptOAuthToken(token)
	if err != nil {
		return fmt.Errorf("error encrypting token: %w", err)
	}

	serialized, err := encryptedToken.Serialize()
	if err != nil {
		return fmt.Errorf("error serializing token: %w", err)
	}

	err = m.store.WithTransactionErr(func(tx db.ExtendQuerier) error {
		at, err := tx.GetAccessTokenByProjectID(ctx, db.GetAccessTokenByProjectIDParams{
			ProjectID: prov.ProjectID,
			Provider:  prov.Name,
		})
		if err != nil {
			return fmt.Errorf("error getting access token: %w", err)
		}

		accessTokenParams := db.UpsertAccessTokenParams{
			ProjectID:       prov.ProjectID,
			Provider:        prov.Name,
			OwnerFilter:     at.OwnerFilter,
			EnrollmentNonce: at.EnrollmentNonce,
			EncryptedAccessToken: pqtype.NullRawMessage{
				RawMessage: serialized,
				Valid:      true,
			},
		}

		_, err = tx.UpsertAccessToken(ctx, accessTokenParams)
		if err != nil {
			return fmt.Errorf("error inserting access token: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error persisting token: %w", err)
	}

	return nil
}

func tokenNeedsRefresh(token oauth2.Token) bool {
	return !token.Valid() || token.Expiry.UTC().Add(tokenExpirationThreshold).Before(time.Now().UTC())
}
