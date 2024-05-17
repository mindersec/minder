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

	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/manager"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// ProviderSessionService is the interface for creating providers from session state
type ProviderSessionService interface {
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
	pErr := providers.ErrProviderNotFoundBy{}
	provider, err := pss.provGetter.GetByName(ctx, stateData.ProjectID, stateData.Provider)
	if errors.As(err, &pErr) {
		createdProvider, err := pss.providerManager.CreateFromConfig(
			ctx, providerClass, stateData.ProjectID, stateData.Provider, stateData.ProviderConfig)
		if err != nil {
			return nil, fmt.Errorf("error creating provider: %w", err)
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
