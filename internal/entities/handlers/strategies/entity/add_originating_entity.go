// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers/manager"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
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
	childProps := properties.NewProperties(entMsg.Entity.GetByProps)

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

		prov, err := a.provMgr.InstantiateFromID(ctx, parentEwp.Entity.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("error getting provider: %w", err)
		}

		upstreamProps, err := prov.FetchAllProperties(ctx, childProps, entMsg.Entity.Type, nil)
		if err != nil {
			return nil, fmt.Errorf("error retrieving properties: %w", err)
		}

		pbEnt, err := prov.PropertiesToProtoMessage(entMsg.Entity.Type, upstreamProps)
		if err != nil {
			return nil, fmt.Errorf("error converting properties to proto message: %w", err)
		}

		legacyId, err := a.upsertLegacyEntity(ctx, entMsg.Entity.Type, parentEwp, pbEnt, t)
		if err != nil {
			return nil, fmt.Errorf("error upserting legacy entity: %w", err)
		}

		childEntName, err := prov.GetEntityName(entMsg.Entity.Type, upstreamProps)
		if err != nil {
			return nil, fmt.Errorf("error getting child entity name: %w", err)
		}

		var entID uuid.UUID
		if legacyId == uuid.Nil {
			// If this isn't backed by a legacy ID we generate a new one
			entID = uuid.New()
		} else {
			// If this represents a legacy entity, we use the legacy ID as the entity ID
			// so we keep the same ID across the system
			entID = legacyId
		}

		childEnt, err := t.CreateOrEnsureEntityByID(ctx, db.CreateOrEnsureEntityByIDParams{
			ID:         entID,
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

		// Persist the properties
		err = a.propSvc.SaveAllProperties(ctx, entID,
			upstreamProps,
			propertyService.CallBuilder().WithStoreOrTransaction(t),
		)
		if err != nil {
			return nil, fmt.Errorf("error persisting properties: %w", err)
		}

		return models.NewEntityWithProperties(childEnt, upstreamProps), nil

	})

	if err != nil {
		return nil, fmt.Errorf("error storing originating entity: %w", err)
	}
	return childEwp, nil
}

// GetName returns the name of the strategy. Used for debugging
func (*addOriginatingEntityStrategy) GetName() string {
	return "addOriginatingEntityStrategy"
}

func (*addOriginatingEntityStrategy) upsertLegacyEntity(
	_ context.Context,
	_ minderv1.Entity,
	_ *models.EntityWithProperties, _ protoreflect.ProtoMessage,
	_ db.ExtendQuerier,
) (uuid.UUID, error) {
	// Legacy entity writes have been removed as part of Phase 1 of the legacy table removal plan.
	// All entities are now written only to entity_instances and properties tables.
	return uuid.Nil, nil
}
