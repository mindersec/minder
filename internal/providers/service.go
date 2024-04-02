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

package providers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/google/go-github/v60/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/github/app"
	"github.com/stacklok/minder/internal/providers/ratecache"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
)

// ProviderService encapsulates methods for creating and updating providers
type ProviderService interface {
	CreateGitHubOAuthProvider(ctx context.Context, providerName string, providerClass db.ProviderClass,
		token oauth2.Token, stateData db.GetProjectIDBySessionStateRow, state string) (*db.Provider, error)
	CreateGitHubAppProvider(ctx context.Context, token oauth2.Token, stateData db.GetProjectIDBySessionStateRow,
		installationID int64, state string) (*db.Provider, error)
	CreateGitHubAppWithoutInvitation(ctx context.Context, token *oauth2.Token,
		installationID int64) (*db.ProviderGithubAppInstallation, error)
	ValidateGitHubInstallationId(ctx context.Context, token *oauth2.Token, installationID int64) error
	DeleteGitHubAppInstallation(ctx context.Context, installationID int64) error
}

// ErrInvalidTokenIdentity is returned when the user identity in the token does not match the expected user identity
// from the state
var ErrInvalidTokenIdentity = errors.New("invalid token identity")

// ProjectFactory may create a project named name for the specified userid if
// present in the system.  If a db.Project is returned, it should be used as the
// location to create a Provider corresponding to the GitHub App installation.
type ProjectFactory func(
	ctx context.Context, qtx db.Querier, name string, user int64) (*db.Project, error)

type providerService struct {
	store           db.Store
	cryptoEngine    crypto.Engine
	mt              metrics.Metrics
	provMt          provtelemetry.ProviderMetrics
	config          *server.ProviderConfig
	projectFactory  ProjectFactory
	restClientCache ratecache.RestClientCache
}

// NewProviderService creates an instance of ProviderService
func NewProviderService(store db.Store, cryptoEngine crypto.Engine, mt metrics.Metrics,
	provMt provtelemetry.ProviderMetrics, config *server.ProviderConfig,
	projectFactory ProjectFactory, restClientCache ratecache.RestClientCache) ProviderService {
	return &providerService{
		store:           store,
		cryptoEngine:    cryptoEngine,
		mt:              mt,
		provMt:          provMt,
		config:          config,
		projectFactory:  projectFactory,
		restClientCache: restClientCache,
	}
}

// CreateGitHubOAuthProvider creates a GitHub OAuth provider with an access token credential
func (p *providerService) CreateGitHubOAuthProvider(
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
		providerDef, err := GetProviderClassDefinition(providerName)
		if err != nil {
			return nil, fmt.Errorf("error getting provider definition: %w", err)
		}

		createdProvider, err := qtx.CreateProvider(ctx, db.CreateProviderParams{
			Name:       providerName,
			ProjectID:  stateData.ProjectID,
			Class:      db.NullProviderClass{ProviderClass: providerClass, Valid: true},
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
func (p *providerService) CreateGitHubAppProvider(
	ctx context.Context,
	token oauth2.Token,
	stateData db.GetProjectIDBySessionStateRow,
	installationID int64,
	state string,
) (*db.Provider, error) {
	installationOwner, err := p.getInstallationOwner(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("error getting installation: %w", err)
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

// CreateGitHubAppWithoutInvitation creates an GitHub App installation that doesn't belong to a project yet
func (p *providerService) CreateGitHubAppWithoutInvitation(
	ctx context.Context,
	token *oauth2.Token,
	installationID int64,
) (*db.ProviderGithubAppInstallation, error) {
	installationOwner, err := p.getInstallationOwner(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("error getting installation: %w", err)
	}

	userID, err := getUserIdFromToken(ctx, token)
	if err != nil || userID == nil || *userID == 0 {
		return nil, fmt.Errorf("error getting user ID from token: %w", err)
	}

	tx, err := p.store.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer p.store.Rollback(tx)
	qtx := p.store.GetQuerierWithTransaction(tx)

	// providerMaker := func(ctx context.Context, qtx db.Querier, projectID uuid.UUID) (db.Provider, error) {
	// 	newProvider, err = createGitHubApp(ctx, qtx, projectID, installationOwner, installationID, nil, sql.NullString{})
	// 	return newProvider, nil
	// }

	projectName := fmt.Sprintf("github-%s", installationOwner.GetLogin())
	project, err := p.projectFactory(ctx, qtx, projectName, *userID /*, providerMaker*/)
	if err != nil {
		// This _can_ be normal if someone enrolls the app without ever logging in to Minder, but should be rare.
		zerolog.Ctx(ctx).Warn().Err(err).Int64("install", installationID).Msg("Error constructing project for install")
		// We couldn't create the project, so create a stand-alone (unclaimed) installation
		gitHubAppInstallation, err := p.store.UpsertInstallationID(ctx, db.UpsertInstallationIDParams{
			ProviderID:        uuid.NullUUID{},
			AppInstallationID: strconv.FormatInt(installationID, 10),
			OrganizationID:    installationOwner.GetID(),
			EnrollingUserID: sql.NullString{
				Valid:  true,
				String: strconv.FormatInt(*userID, 10),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("error saving installation ID: %w", err)
		}
		return &gitHubAppInstallation, nil
	}

	provider, err := createGitHubApp(ctx, qtx, project.ID, installationOwner, installationID, nil, sql.NullString{})
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub App Provider: %w", err)

	}

	if err = p.store.Commit(tx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	install, err := p.store.GetInstallationIDByProviderID(ctx, uuid.NullUUID{
		UUID:  provider.ID,
		Valid: true,
	})

	return &install, err
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
		Class:      db.NullProviderClass{ProviderClass: db.ProviderClassGithubApp, Valid: true},
		Implements: app.Implements,
		Definition: json.RawMessage(`{"github-app": {}}`),
		AuthFlows:  app.AuthorizationFlows,
	})
	if err != nil {
		return db.Provider{}, err
	}

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
		AppInstallationID: strconv.FormatInt(installationID, 10),
		EnrollmentNonce:   nonce,
	})
	if err != nil {
		return db.Provider{}, err
	}
	return savedProvider, nil
}

// ValidateGitHubInstallationId checks if the user has access to the installation ID
func (_ *providerService) ValidateGitHubInstallationId(ctx context.Context, token *oauth2.Token, installationID int64) error {
	// Get the installations this user has access to
	ghClient := github.NewClient(nil).WithAuthToken(token.AccessToken)

	installations, _, err := ghClient.Apps.ListUserInstallations(ctx, nil)
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

func (p *providerService) DeleteGitHubAppInstallation(ctx context.Context, installationID int64) error {
	installation, err := p.store.GetInstallationIDByAppID(ctx, strconv.FormatInt(installationID, 10))
	if err != nil {
		return fmt.Errorf("error getting installation: %w", err)
	}

	if installation.ProviderID.UUID == uuid.Nil {
		zerolog.Ctx(ctx).Info().
			Int64("installationID", installationID).
			Msg("Installation not claimed, deleting the installation")
		return p.store.DeleteInstallationIDByAppID(ctx, strconv.FormatInt(installationID, 10))
	}

	zerolog.Ctx(ctx).Info().
		Int64("installationID", installationID).
		Str("providerID", installation.ProviderID.UUID.String()).
		Msg("Deleting claimed installation")
	return p.store.DeleteProvider(ctx, installation.ProviderID.UUID)
}

func (p *providerService) verifyProviderTokenIdentity(
	ctx context.Context, stateData db.GetProjectIDBySessionStateRow, provider db.Provider, token string) error {
	pbOpts := []ProviderBuilderOption{
		WithProviderMetrics(p.provMt),
		WithRestClientCache(p.restClientCache),
	}
	builder := NewProviderBuilder(&provider, sql.NullString{}, credentials.NewGitHubTokenCredential(token),
		p.config, pbOpts...)
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

func (p *providerService) getInstallationOwner(ctx context.Context, installationID int64) (*github.User, error) {
	privateKey, err := p.config.GitHubApp.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error getting GitHub App private key: %w", err)
	}
	jwt, err := credentials.CreateGitHubAppJWT(p.config.GitHubApp.AppID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub App JWT: %w", err)
	}

	ghClient := github.NewClient(nil).WithAuthToken(jwt)
	installation, _, err := ghClient.Apps.GetInstallation(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("error getting installation: %w", err)
	}
	return installation.GetAccount(), nil
}

func getUserIdFromToken(ctx context.Context, token *oauth2.Token) (*int64, error) {
	ghClient := github.NewClient(nil).WithAuthToken(token.AccessToken)

	user, _, err := ghClient.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	return user.ID, nil
}
