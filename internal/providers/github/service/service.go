//
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

// Package service contains the github
package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/google/go-github/v61/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/credentials"
	github2 "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/github/app"
	"github.com/stacklok/minder/internal/providers/ratecache"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
)

// GitHubProviderService encapsulates methods for creating and updating providers
type GitHubProviderService interface {
	// CreateGitHubOAuthProvider creates a GitHub OAuth provider with an access token credential
	CreateGitHubOAuthProvider(ctx context.Context, providerName string, providerClass db.ProviderClass,
		token oauth2.Token, stateData db.GetProjectIDBySessionStateRow, state string) (*db.Provider, error)
	// CreateGitHubAppProvider creates a GitHub App provider with an installation ID in a known project
	CreateGitHubAppProvider(ctx context.Context, token oauth2.Token, stateData db.GetProjectIDBySessionStateRow,
		installationID int64, state string) (*db.Provider, error)
	// CreateGitHubAppWithoutInvitation either creates a new project for the selected app, or stores
	// the installation in preparation for creating a new project when the authorizing user logs in.
	//
	// Note that this function may return nil, nil if the installation user is not known to Minder.
	CreateGitHubAppWithoutInvitation(ctx context.Context, qtx db.Querier, userID int64,
		installationID int64) (*db.Project, error)
	// ValidateGitHubInstallationId checks if the supplied GitHub token has access to the installation ID
	ValidateGitHubInstallationId(ctx context.Context, token *oauth2.Token, installationID int64) error
	// DeleteGitHubAppInstallation deletes the GitHub App installation and provider from the database.
	DeleteGitHubAppInstallation(ctx context.Context, installationID int64) error
	// ValidateGitHubAppWebhookPayload validates the payload of a GitHub App webhook.
	ValidateGitHubAppWebhookPayload(r *http.Request) (payload []byte, err error)
	DeleteProvider(ctx context.Context, provider *db.Provider) error
}

// TypeGitHubOrganization is the type returned from the GitHub API when the owner is an organization
const TypeGitHubOrganization = "Organization"

// ErrInvalidTokenIdentity is returned when the user identity in the token does not match the expected user identity
// from the state
var ErrInvalidTokenIdentity = errors.New("invalid token identity")

// ProjectFactory may create a project named name for the specified userid if
// present in the system.  If a db.Project is returned, it should be used as the
// location to create a Provider corresponding to the GitHub App installation.
type ProjectFactory func(
	ctx context.Context, qtx db.Querier, name string, user int64) (*db.Project, error)

type ghProviderService struct {
	store               db.Store
	cryptoEngine        crypto.Engine
	mt                  metrics.Metrics
	provMt              provtelemetry.ProviderMetrics
	config              *server.ProviderConfig
	projectFactory      ProjectFactory
	restClientCache     ratecache.RestClientCache
	ghClientService     github2.ClientService
	fallbackTokenClient *github.Client
}

// NewGithubProviderService creates an instance of GitHubProviderService
func NewGithubProviderService(
	store db.Store,
	cryptoEngine crypto.Engine,
	mt metrics.Metrics,
	provMt provtelemetry.ProviderMetrics,
	config *server.ProviderConfig,
	projectFactory ProjectFactory,
	restClientCache ratecache.RestClientCache,
	fallbackTokenClient *github.Client,
) GitHubProviderService {
	return &ghProviderService{
		store:               store,
		cryptoEngine:        cryptoEngine,
		mt:                  mt,
		provMt:              provMt,
		config:              config,
		projectFactory:      projectFactory,
		restClientCache:     restClientCache,
		ghClientService:     github2.ClientServiceImplementation{},
		fallbackTokenClient: fallbackTokenClient,
	}
}

// CreateGitHubOAuthProvider creates a GitHub OAuth provider with an access token credential
func (p *ghProviderService) CreateGitHubOAuthProvider(
	ctx context.Context,
	providerName string,
	providerClass db.ProviderClass,
	token oauth2.Token,
	stateData db.GetProjectIDBySessionStateRow,
	state string,
) (*db.Provider, error) {
	tx, err := p.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error starting transaction: %v", err)
	}
	defer p.store.Rollback(tx)

	qtx := p.store.GetQuerierWithTransaction(tx)

	// Check if the provider exists
	provider, err := qtx.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:     providerName,
		Projects: []uuid.UUID{stateData.ProjectID},
	})
	if errors.Is(err, sql.ErrNoRows) {

		// If the provider does not exist, create it
		providerDef, err := providers.GetProviderClassDefinition(providerName)
		if err != nil {
			return nil, fmt.Errorf("error getting provider definition: %w", err)
		}

		createdProvider, err := qtx.CreateProvider(ctx, db.CreateProviderParams{
			Name:       providerName,
			ProjectID:  stateData.ProjectID,
			Class:      providerClass,
			Implements: providerDef.Traits,
			Definition: json.RawMessage(`{"github": {}}`),
			AuthFlows:  providerDef.AuthorizationFlows,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating provider: %w", err)
		}
		provider = createdProvider
	} else if err != nil {
		return nil, fmt.Errorf("error getting provider from DB: %w", err)
	}

	// Older enrollments may not have a RemoteUser stored; these should age out fairly quickly.
	p.mt.AddTokenOpCount(ctx, "check", stateData.RemoteUser.Valid)
	if stateData.RemoteUser.Valid {
		if err := p.verifyProviderTokenIdentity(ctx, stateData, provider, token.AccessToken); err != nil {
			return nil, ErrInvalidTokenIdentity
		}
	} else {
		zerolog.Ctx(ctx).Warn().Msg("RemoteUser not found in session state")
	}

	ftoken := &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: "",
	}

	// Convert token to JSON
	jsonData, err := json.Marshal(ftoken)
	if err != nil {
		return nil, fmt.Errorf("error marshaling token: %w", err)
	}

	// encode token
	encryptedToken, err := p.cryptoEngine.EncryptOAuthToken(jsonData)
	if err != nil {
		return nil, fmt.Errorf("error encoding token: %w", err)
	}

	encodedToken := base64.StdEncoding.EncodeToString(encryptedToken)

	_, err = qtx.UpsertAccessToken(ctx, db.UpsertAccessTokenParams{
		ProjectID:      stateData.ProjectID,
		Provider:       providerName,
		EncryptedToken: encodedToken,
		OwnerFilter:    stateData.OwnerFilter,
		EnrollmentNonce: sql.NullString{
			Valid:  true,
			String: state,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error inserting access token: %w", err)
	}
	if err := p.store.Commit(tx); err != nil {

		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}
	return &provider, nil
}

// CreateGitHubAppProvider creates a GitHub App provider with an installation ID
func (p *ghProviderService) CreateGitHubAppProvider(
	ctx context.Context,
	token oauth2.Token,
	stateData db.GetProjectIDBySessionStateRow,
	installationID int64,
	state string,
) (*db.Provider, error) {
	installationOwner, err := p.getInstallationOwner(ctx, installationID)
	if err != nil {
		return nil, err
	}

	return db.WithTransaction(p.store, func(qtx db.ExtendQuerier) (*db.Provider, error) {
		validateOwnership := func(ctx context.Context, provider db.Provider) error {
			// Older enrollments may not have a RemoteUser stored; these should age out fairly quickly.
			p.mt.AddTokenOpCount(ctx, "check", stateData.RemoteUser.Valid)
			if stateData.RemoteUser.Valid {
				if err := p.verifyProviderTokenIdentity(ctx, stateData, provider, token.AccessToken); err != nil {
					return ErrInvalidTokenIdentity
				}
			} else {
				zerolog.Ctx(ctx).Warn().Msg("RemoteUser not found in session state")
			}
			return nil
		}

		provider, err := createGitHubApp(
			ctx,
			qtx,
			stateData.ProjectID,
			installationOwner,
			installationID,
			validateOwnership,
			sql.NullString{
				String: state,
				Valid:  true,
			},
		)

		return &provider, err
	})
}

// CreateGitHubAppWithoutInvitation either creates a new project for the selected app installation, or stores
// it in preparation for creating a new project when the authorizing user logs in.
//
// Note that this function may return nil, nil if the installation user is not known to Minder.
func (p *ghProviderService) CreateGitHubAppWithoutInvitation(
	ctx context.Context,
	qtx db.Querier,
	userID int64,
	installationID int64,
) (*db.Project, error) {
	installationOwner, err := p.getInstallationOwner(ctx, installationID)
	if err != nil {
		return nil, err
	}

	isOrg := installationOwner.GetType() == TypeGitHubOrganization
	projectName := fmt.Sprintf("github-%s", installationOwner.GetLogin())
	project, err := p.projectFactory(ctx, qtx, projectName, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			// This _can_ be normal if someone enrolls the app without ever logging in to Minder, but should be rare.
			zerolog.Ctx(ctx).Warn().Err(err).Int64("install", installationID).Msg("No user for install, creating unclaimed record")
		} else {
			zerolog.Ctx(ctx).Warn().Err(err).Int64("install", installationID).Msg("Error constructing project for install")
		}
		// We couldn't create the project, so create a stand-alone (unclaimed) installation
		_, err := p.store.UpsertInstallationID(ctx, db.UpsertInstallationIDParams{
			ProviderID:        uuid.NullUUID{},
			AppInstallationID: installationID,
			OrganizationID:    installationOwner.GetID(),
			EnrollingUserID: sql.NullString{
				Valid:  true,
				String: strconv.FormatInt(userID, 10),
			},
			IsOrg: isOrg,
		})
		if err != nil {
			return nil, fmt.Errorf("error saving installation ID: %w", err)
		}
		return nil, nil
	}

	zerolog.Ctx(ctx).Info().Str("project", project.ID.String()).Int64("owner", installationOwner.GetID()).
		Msg("Creating GitHub App Provider")

	_, err = createGitHubApp(ctx, qtx, project.ID, installationOwner, installationID, nil, sql.NullString{})
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub App Provider: %w", err)

	}

	return project, err
}

// Internal shared implementation between CreateGitHubAppProvider and CreateGitHubAppWithoutInvitation.
// Note that this does not validate the projectId, and assumes the caller does so!
func createGitHubApp(
	ctx context.Context,
	qtx db.Querier,
	projectId uuid.UUID,
	installationOwner *github.User,
	installationID int64,
	validateOwnership func(ctx context.Context, provider db.Provider) error,
	nonce sql.NullString,
) (db.Provider, error) {
	// Save the installation ID and create a provider
	savedProvider, err := qtx.CreateProvider(ctx, db.CreateProviderParams{
		Name:       fmt.Sprintf("%s-%s", db.ProviderClassGithubApp, installationOwner.GetLogin()),
		ProjectID:  projectId,
		Class:      db.ProviderClassGithubApp,
		Implements: app.Implements,
		Definition: json.RawMessage(`{"github-app": {}}`),
		AuthFlows:  app.AuthorizationFlows,
	})
	if err != nil {
		return db.Provider{}, err
	}

	isOrg := installationOwner.GetType() == TypeGitHubOrganization

	if validateOwnership != nil {
		// TODO: it would be nice if validateOwnership didn't need to use a provider to get a
		// github.Client, because then we could call it _before_ createGitHubApp, rather than
		// in the middle...
		if err := validateOwnership(ctx, savedProvider); err != nil {
			return db.Provider{}, err
		}
	}

	_, err = qtx.UpsertInstallationID(ctx, db.UpsertInstallationIDParams{
		ProviderID: uuid.NullUUID{
			UUID:  savedProvider.ID,
			Valid: true,
		},
		ProjectID: uuid.NullUUID{
			UUID:  projectId,
			Valid: true,
		},
		OrganizationID:    installationOwner.GetID(),
		AppInstallationID: installationID,
		EnrollmentNonce:   nonce,
		IsOrg:             isOrg,
	})
	if err != nil {
		return db.Provider{}, err
	}
	return savedProvider, nil
}

// ValidateGitHubInstallationId checks if the user has access to the installation ID
func (p *ghProviderService) ValidateGitHubInstallationId(ctx context.Context, token *oauth2.Token, installationID int64) error {
	installations, err := p.ghClientService.ListUserInstallations(ctx, token)
	if err != nil {
		return fmt.Errorf("error getting user installations: %w", err)
	}

	matchesID := func(installation *github.Installation) bool {
		return installation.GetID() == installationID
	}

	i := slices.IndexFunc(installations, matchesID)
	if i == -1 {
		// The user does not have access to the installation
		return fmt.Errorf("user does not have access to installation ID %d", installationID)
	}

	return nil
}

// GitHubAppInstallationDeletedPayload represents the payload of a GitHub App installation deleted event
type GitHubAppInstallationDeletedPayload struct {
	InstallationID int64 `json:"installation_id"`
}

func (p *ghProviderService) DeleteGitHubAppInstallation(ctx context.Context, installationID int64) error {
	installation, err := p.store.GetInstallationIDByAppID(ctx, installationID)
	if err != nil {
		// This installation has already been deleted
		if errors.Is(err, sql.ErrNoRows) {
			zerolog.Ctx(ctx).Info().
				Int64("installationID", installationID).
				Msg("Installation already deleted")
			return nil
		}
		return fmt.Errorf("error getting installation: %w", err)
	}

	if installation.ProviderID.UUID == uuid.Nil {
		zerolog.Ctx(ctx).Info().
			Int64("installationID", installationID).
			Msg("Installation not claimed, deleting the installation")
		return p.store.DeleteInstallationIDByAppID(ctx, installationID)
	}

	zerolog.Ctx(ctx).Info().
		Int64("installationID", installationID).
		Str("providerID", installation.ProviderID.UUID.String()).
		Msg("Deleting claimed installation")
	return p.store.DeleteProvider(ctx, db.DeleteProviderParams{
		ID:        installation.ProviderID.UUID,
		ProjectID: installation.ProjectID.UUID,
	})
}

func (p *ghProviderService) ValidateGitHubAppWebhookPayload(r *http.Request) (payload []byte, err error) {
	secret, err := p.config.GitHubApp.GetWebhookSecret()
	if err != nil {
		return nil, err
	}
	return github.ValidatePayload(r, []byte(secret))
}

// DeleteProvider deletes a provider and removes the credentials from the downstream service, if possible
func (p *ghProviderService) DeleteProvider(ctx context.Context, provider *db.Provider) error {
	// Uninstall the GitHub App if it's a GitHub App provider
	err := p.deleteInstallation(ctx, provider.ID)
	if err != nil {
		return fmt.Errorf("error deleting GitHub App installation: %w", err)
	}

	return p.store.DeleteProvider(ctx, db.DeleteProviderParams{ID: provider.ID, ProjectID: provider.ProjectID})
}

// deleteInstallation deletes the installation from GitHub, if the provider has an associated installation
func (p *ghProviderService) deleteInstallation(ctx context.Context, providerID uuid.UUID) error {
	installation, err := p.store.GetInstallationIDByProviderID(ctx, uuid.NullUUID{
		UUID:  providerID,
		Valid: true,
	})

	// If there are no associated installations, return early
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	} else if err != nil {
		return fmt.Errorf("error getting installation: %w", err)
	}

	privateKey, err := p.config.GitHubApp.GetPrivateKey()
	if err != nil {
		return fmt.Errorf("error getting GitHub App private key: %w", err)
	}
	jwt, err := credentials.CreateGitHubAppJWT(p.config.GitHubApp.AppID, privateKey)
	if err != nil {
		return fmt.Errorf("error creating GitHub App JWT: %w", err)
	}

	resp, err := p.ghClientService.DeleteInstallation(ctx, installation.AppInstallationID, jwt)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the installation is not found, we can ignore the error, user might have deleted it manually
			return nil
		}
		return fmt.Errorf("error deleting installation: %w", err)
	}
	return nil
}

func (p *ghProviderService) verifyProviderTokenIdentity(
	ctx context.Context, stateData db.GetProjectIDBySessionStateRow, provider db.Provider, token string) error {
	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(p.provMt),
		providers.WithRestClientCache(p.restClientCache),
	}
	builder := providers.NewProviderBuilder(&provider, sql.NullString{}, false, credentials.NewGitHubTokenCredential(token),
		p.config, p.fallbackTokenClient, pbOpts...)
	// NOTE: this is github-specific at the moment.  We probably need to generally
	// re-think token enrollment when we add more providers.
	ghClient, err := builder.GetGitHub()
	if err != nil {
		return fmt.Errorf("error creating GitHub client: %w", err)
	}
	userId, err := ghClient.GetUserId(ctx)
	if err != nil {
		return fmt.Errorf("error getting user ID: %w", err)
	}
	if strconv.FormatInt(userId, 10) != stateData.RemoteUser.String {
		return fmt.Errorf("user ID mismatch: %d != %s", userId, stateData.RemoteUser.String)
	}
	return nil
}

func (p *ghProviderService) getInstallationOwner(ctx context.Context, installationID int64) (*github.User, error) {
	privateKey, err := p.config.GitHubApp.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error getting GitHub App private key: %w", err)
	}
	jwt, err := credentials.CreateGitHubAppJWT(p.config.GitHubApp.AppID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub App JWT: %w", err)
	}

	installation, _, err := p.ghClientService.GetInstallation(ctx, installationID, jwt)
	if err != nil {
		return nil, fmt.Errorf("error getting installation: %w", err)
	}
	return installation.GetAccount(), nil
}
