// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package session contains the business logic for creating providers from session state.
package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers"
	"github.com/mindersec/minder/internal/providers/manager"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// ProviderSessionService is the interface for creating providers from session state
type ProviderSessionService interface {
	// CreateProviderFromSessionState creates a provider from the session
	// state, storing the encrypted credentials as its access token. If the
	// provider already exists, it is returned and the token is refreshed.
	// If creation loses a race against a concurrent enrollment, the existing
	// provider is returned together with the unique-violation error and no
	// access token is stored; callers can detect that case with
	// db.ErrIsUniqueViolation and continue with the returned provider.
	CreateProviderFromSessionState(
		ctx context.Context, providerClass db.ProviderClass,
		encryptedCreds *crypto.EncryptedData, state string,
	) (*db.Provider, error)
}

type providerByNameGetter interface {
	GetByName(ctx context.Context, projectID uuid.UUID, name string) (*db.Provider, error)
}

type dbSessionStore interface {
	GetProjectIDBySessionState(ctx context.Context, sessionState string) (db.GetProjectIDBySessionStateRow, error)
	UpsertAccessToken(ctx context.Context, arg db.UpsertAccessTokenParams) (db.ProviderAccessToken, error)
}

type providerSessionService struct {
	providerManager manager.ProviderManager
	provGetter      providerByNameGetter
	dbStore         dbSessionStore
}

// NewProviderSessionService creates a new provider session service
func NewProviderSessionService(
	providerManager manager.ProviderManager,
	provGetter providerByNameGetter,
	dbStore dbSessionStore,
) ProviderSessionService {
	return &providerSessionService{
		providerManager: providerManager,
		provGetter:      provGetter,
		dbStore:         dbStore,
	}
}

func (pss *providerSessionService) CreateProviderFromSessionState(
	ctx context.Context, providerClass db.ProviderClass,
	encryptedCreds *crypto.EncryptedData, state string,
) (*db.Provider, error) {
	stateData, err := pss.dbStore.GetProjectIDBySessionState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("error getting state data by session state: %w", err)
	}

	serialized, err := encryptedCreds.Serialize()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error serializing secret: %s", err)
	}

	accessTokenParams := db.UpsertAccessTokenParams{
		ProjectID:       stateData.ProjectID,
		Provider:        stateData.Provider,
		OwnerFilter:     stateData.OwnerFilter,
		EnrollmentNonce: sql.NullString{String: state, Valid: true},
		EncryptedAccessToken: pqtype.NullRawMessage{
			RawMessage: serialized,
			Valid:      true,
		},
	}

	// Check if the provider exists
	provider, err := pss.provGetter.GetByName(ctx, stateData.ProjectID, stateData.Provider)
	if _, notFound := errors.AsType[providers.ErrProviderNotFoundBy](err); notFound {
		createdProvider, createErr := pss.providerManager.CreateFromConfig(
			ctx, providerClass, stateData.ProjectID, stateData.Provider, stateData.ProviderConfig)
		if db.ErrIsUniqueViolation(createErr) {
			// We lost the creation race against a concurrent enrollment of
			// the same provider. Return the winner's provider together with
			// the uniqueness error so callers can detect the race; the
			// winning flow stores its own access token, so we do not store
			// a second one here.
			existing, getErr := pss.provGetter.GetByName(ctx, stateData.ProjectID, stateData.Provider)
			if getErr != nil {
				return nil, fmt.Errorf("error getting provider from DB after losing creation race: %w", getErr)
			}
			return existing, createErr
		}
		if createErr != nil {
			return nil, fmt.Errorf("error creating provider: %w", createErr)
		}
		provider = createdProvider
	} else if err != nil {
		return nil, fmt.Errorf("error getting provider from DB: %w", err)
	}

	_, err = pss.dbStore.UpsertAccessToken(ctx, accessTokenParams)
	if err != nil {
		return nil, fmt.Errorf("error inserting access token: %w", err)
	}

	return provider, nil
}
