// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"fmt"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	entityService "github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/providers/manager"
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
	// Note: These reads are outside the transaction boundary in EntityCreator.CreateEntity
	// because they read stable data (parent entity and provider configuration).
	// The transaction in CreateEntity protects the writes (entity + properties).
	// If there's a race where parent is deleted between read/write, the FK constraint catches it.
	provider, err := a.store.GetProviderByID(ctx, parentEwp.Entity.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("error getting provider: %w", err)
	}

	// Use EntityCreator to create child entity
	// Note: Child entities (artifacts, releases, PRs) don't trigger reconciliation events.
	// This matches existing behavior and avoids potential loops since this code runs
	// from a message handler. The parent repository's reconciliation handles the
	// evaluation of child entities through the entity evaluation graph.
	childEwp, err := a.entityCreator.CreateEntity(ctx, &provider,
		parentEwp.Entity.ProjectID, entMsg.Entity.Type, childProps,
		&entityService.EntityCreationOptions{
			OriginatingEntityID:        &parentEwp.Entity.ID,
			RegisterWithProvider:       false, // No webhooks for child entities
			PublishReconciliationEvent: false, // Explained above
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
