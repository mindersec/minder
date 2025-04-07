// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers/manager"
)

type refreshEntityByIDStrategy struct {
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
	store   db.Store
}

// NewRefreshEntityByIDStrategy creates a new getEntityByUpstreamIDStrategy.
func NewRefreshEntityByIDStrategy(
	propSvc propertyService.PropertiesService,
	provManager manager.ProviderManager,
	store db.Store,
) strategies.GetEntityStrategy {
	return &refreshEntityByIDStrategy{
		provMgr: provManager,
		propSvc: propSvc,
		store:   store,
	}
}

// GetEntity gets an entity by its upstream ID.
func (r *refreshEntityByIDStrategy) GetEntity(
	ctx context.Context, entMsg *message.HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	if entMsg.Entity.EntityID == uuid.Nil {
		return nil, fmt.Errorf("entity id is nil")
	}

	getEnt, err := db.WithTransaction(r.store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
		ewp, err := r.propSvc.EntityWithPropertiesByID(
			ctx, entMsg.Entity.EntityID,
			propertyService.CallBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error getting entity: %w", err)
		}

		err = r.propSvc.RetrieveAllPropertiesForEntity(
			ctx, ewp, r.provMgr,
			propertyService.ReadBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error retrieving properties for entity: %w", err)
		}
		return ewp, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error refreshing entity: %w", err)
	}

	return getEnt, nil
}

// GetName returns the name of the strategy. Used for debugging
func (*refreshEntityByIDStrategy) GetName() string {
	return "getEntityByID"
}
