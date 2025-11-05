// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	entityService "github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/providers/manager"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

type addOriginatingEntityStrategy struct {
	propSvc       propertyService.PropertiesService
	provMgr       manager.ProviderManager
	store         db.Store
	entityCreator entityService.EntityCreator
}

// NewAddOriginatingEntityStrategy creates a new addOriginatingEntityStrategy.
func NewAddOriginatingEntityStrategy(
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
	entityCreator entityService.EntityCreator,
) strategies.GetEntityStrategy {
	return &addOriginatingEntityStrategy{
		propSvc:       propSvc,
		provMgr:       provMgr,
		store:         store,
		entityCreator: entityCreator,
	}
}

// GetEntity adds an originating entity.
func (a *addOriginatingEntityStrategy) GetEntity(
	ctx context.Context, entMsg *message.HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	childProps := properties.NewProperties(entMsg.Entity.GetByProps)

	// Get parent entity (originator)
	parentEwp, err := getEntityInner(
		ctx,
		entMsg.Originator.Type, entMsg.Originator.GetByProps, entMsg.Hint,
		a.propSvc,
		nil)
	if err != nil {
		return nil, fmt.Errorf("error getting parent entity: %w", err)
	}

	// Get provider from DB
	provider, err := a.store.GetProviderByID(ctx, parentEwp.Entity.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("error getting provider: %w", err)
	}

	// Use EntityCreator to create child entity
	childEwp, err := a.entityCreator.CreateEntity(ctx, &provider,
		parentEwp.Entity.ProjectID, entMsg.Entity.Type, childProps,
		&entityService.EntityCreationOptions{
			OriginatingEntityID:        &parentEwp.Entity.ID,
			RegisterWithProvider:       false, // No webhooks for child entities
			PublishReconciliationEvent: false, // No reconciliation for child entities
		})
	if err != nil {
		return nil, fmt.Errorf("error creating entity: %w", err)
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
