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
	"github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	"github.com/stacklok/minder/internal/entities/models"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/providers/manager"
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
func (_ *refreshEntityByIDStrategy) GetName() string {
	return "getEntityByID"
}
