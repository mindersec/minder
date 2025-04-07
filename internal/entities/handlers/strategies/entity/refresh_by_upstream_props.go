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
	"github.com/mindersec/minder/internal/providers/manager"
)

type refreshEntityByUpstreamIDStrategy struct {
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
	store   db.Store
}

// NewRefreshEntityByUpstreamPropsStrategy creates a new refreshEntityByUpstreamIDStrategy.
func NewRefreshEntityByUpstreamPropsStrategy(
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
) strategies.GetEntityStrategy {
	return &refreshEntityByUpstreamIDStrategy{
		propSvc: propSvc,
		provMgr: provMgr,
		store:   store,
	}
}

// GetEntity refreshes an entity by its upstream ID.
func (r *refreshEntityByUpstreamIDStrategy) GetEntity(
	ctx context.Context, entMsg *message.HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	getEnt, err := db.WithTransaction(r.store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
		ewp, err := getEntityInner(
			ctx,
			entMsg.Entity.Type, entMsg.Entity.GetByProps, entMsg.Hint,
			r.propSvc, propertyService.CallBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error getting entity: %w", err)
		}

		err = r.propSvc.RetrieveAllPropertiesForEntity(
			ctx, ewp, r.provMgr,
			propertyService.ReadBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error fetching entity: %w", err)
		}
		return ewp, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error refreshing entity: %w", err)
	}

	return getEnt, nil
}

// GetName returns the name of the strategy. Used for debugging
func (*refreshEntityByUpstreamIDStrategy) GetName() string {
	return "refreshEntityByUpstreamIDStrategy"
}
