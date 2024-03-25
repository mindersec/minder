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
	"strconv"

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
	"github.com/stacklok/minder/internal/providers/ratecache"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
)

// ProviderService encapsulates methods for creating and updating providers
type ProviderService interface {
	CreateGitHubOAuthProvider(ctx context.Context, providerName string, providerClass db.ProviderClass,
		token oauth2.Token, stateData db.GetProjectIDBySessionStateRow) (*db.Provider, error)
}

// ErrInvalidTokenIdentity is returned when the user identity in the token does not match the expected user identity
// from the state
var ErrInvalidTokenIdentity = errors.New("invalid token identity")

type providerService struct {
	store           db.Store
	cryptoEngine    crypto.Engine
	mt              metrics.Metrics
	provMt          provtelemetry.ProviderMetrics
	config          *server.ProviderConfig
	restClientCache ratecache.RestClientCache
}

// NewProviderService creates an instance of ProviderService
func NewProviderService(store db.Store, cryptoEngine crypto.Engine, mt metrics.Metrics,
	provMt provtelemetry.ProviderMetrics, config *server.ProviderConfig, restClientCache ratecache.RestClientCache) ProviderService {
	return &providerService{
		store:           store,
		cryptoEngine:    cryptoEngine,
		mt:              mt,
		provMt:          provMt,
		config:          config,
		restClientCache: restClientCache,
	}
}

// CreateGitHubOAuthProvider creates a GitHub OAuth provider with an access token credential
func (p *providerService) CreateGitHubOAuthProvider(ctx context.Context, providerName string, providerClass db.ProviderClass,
	token oauth2.Token, stateData db.GetProjectIDBySessionStateRow) (*db.Provider, error) {
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
	})
	if err != nil {
		return nil, fmt.Errorf("error inserting access token: %w", err)
	}
	if err := p.store.Commit(tx); err != nil {

		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}
	return &provider, nil
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
