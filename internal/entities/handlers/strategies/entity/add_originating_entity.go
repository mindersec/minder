// Copyright 2024 Stacklok, Inc.
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

// Package entity contains the entity creation strategies
package entity

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	"github.com/stacklok/minder/internal/providers/manager"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type addOriginatingEntityStrategy struct {
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
	store   db.Store
}

// NewAddOriginatingEntityStrategy creates a new addOriginatingEntityStrategy.
func NewAddOriginatingEntityStrategy(
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
) strategies.GetEntityStrategy {
	return &addOriginatingEntityStrategy{
		propSvc: propSvc,
		provMgr: provMgr,
		store:   store,
	}
}

// GetEntity adds an originating entity.
func (a *addOriginatingEntityStrategy) GetEntity(
	ctx context.Context, entMsg *message.HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	childProps, err := properties.NewProperties(entMsg.Entity.GetByProps)
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

	// store the originating entity
	childEwp, err := db.WithTransaction(a.store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
		parentEwp, err := getEntityInner(
			ctx,
			entMsg.Originator.Type, entMsg.Originator.GetByProps, entMsg.Hint,
			a.propSvc,
			propertyService.CallBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error getting parent entity: %w", err)
		}

		legacyId, err := a.upsertLegacyEntity(ctx, entMsg.Entity.Type, parentEwp, childProps, t)
		if err != nil {
			return nil, fmt.Errorf("error upserting legacy entity: %w", err)
		}

		prov, err := a.provMgr.InstantiateFromID(ctx, parentEwp.Entity.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("error getting provider: %w", err)
		}

		childEntName, err := prov.GetEntityName(entMsg.Entity.Type, childProps)
		if err != nil {
			return nil, fmt.Errorf("error getting child entity name: %w", err)
		}

		childEnt, err := t.CreateOrEnsureEntityByID(ctx, db.CreateOrEnsureEntityByIDParams{
			ID:         legacyId,
			EntityType: entities.EntityTypeToDB(entMsg.Entity.Type),
			Name:       childEntName,
			ProjectID:  parentEwp.Entity.ProjectID,
			ProviderID: parentEwp.Entity.ProviderID,
			OriginatedFrom: uuid.NullUUID{
				UUID:  parentEwp.Entity.ID,
				Valid: true,
			},
		})
		if err != nil {
			return nil, err
		}

		upstreamProps, err := a.propSvc.RetrieveAllProperties(ctx, prov,
			parentEwp.Entity.ProjectID, parentEwp.Entity.ProviderID,
			childProps, entMsg.Entity.Type,
			propertyService.ReadBuilder().WithStoreOrTransaction(t),
		)
		if err != nil {
			return nil, fmt.Errorf("error retrieving properties: %w", err)
		}

		return models.NewEntityWithProperties(childEnt, upstreamProps), nil

	})

	if err != nil {
		return nil, fmt.Errorf("error storing originating entity: %w", err)
	}
	return childEwp, nil
}

// GetName returns the name of the strategy. Used for debugging
func (_ *addOriginatingEntityStrategy) GetName() string {
	return "addOriginatingEntityStrategy"
}

func (_ *addOriginatingEntityStrategy) upsertLegacyEntity(
	ctx context.Context,
	entType minderv1.Entity,
	parentEwp *models.EntityWithProperties, childProps *properties.Properties,
	t db.ExtendQuerier,
) (uuid.UUID, error) {
	var legacyId uuid.UUID

	switch entType { // nolint:exhaustive
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		dbPr, err := t.UpsertPullRequest(ctx, db.UpsertPullRequestParams{
			RepositoryID: parentEwp.Entity.ID,
			PrNumber:     childProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("error upserting pull request: %w", err)
		}
		legacyId = dbPr.ID
	case minderv1.Entity_ENTITY_ARTIFACTS:
		// TODO: remove this once we migrate artifacts to entities. We should get rid of the provider name.
		dbProv, err := t.GetProviderByID(ctx, parentEwp.Entity.ProviderID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("error getting provider: %w", err)
		}

		dbArtifact, err := t.UpsertArtifact(ctx, db.UpsertArtifactParams{
			RepositoryID: uuid.NullUUID{
				UUID:  parentEwp.Entity.ID,
				Valid: true,
			},
			ArtifactName:       childProps.GetProperty(ghprop.ArtifactPropertyName).GetString(),
			ArtifactType:       childProps.GetProperty(ghprop.ArtifactPropertyType).GetString(),
			ArtifactVisibility: childProps.GetProperty(ghprop.ArtifactPropertyVisibility).GetString(),
			ProjectID:          parentEwp.Entity.ProjectID,
			ProviderID:         parentEwp.Entity.ProviderID,
			ProviderName:       dbProv.Name,
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("error upserting artifact: %w", err)
		}
		legacyId = dbArtifact.ID
	}

	return legacyId, nil
}
