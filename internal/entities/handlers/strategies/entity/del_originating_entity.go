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
	"database/sql"
	"errors"
	"fmt"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	"github.com/stacklok/minder/internal/entities/models"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/providers/manager"
)

type delOriginatingEntityStrategy struct {
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
	store   db.Store
}

// NewDelOriginatingEntityStrategy creates a new delOriginatingEntityStrategy.
func NewDelOriginatingEntityStrategy(
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
) strategies.GetEntityStrategy {
	return &delOriginatingEntityStrategy{
		propSvc: propSvc,
		provMgr: provMgr,
		store:   store,
	}
}

// GetEntity deletes the originating entity.
func (d *delOriginatingEntityStrategy) GetEntity(
	ctx context.Context, entMsg *message.HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	tx, err := d.store.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		_ = d.store.Rollback(tx)
	}()

	txq := d.store.GetQuerierWithTransaction(tx)
	if txq == nil {
		return nil, fmt.Errorf("error getting querier")
	}

	_, err = getEntityInner(
		ctx,
		entMsg.Originator.Type, entMsg.Originator.GetByProps, entMsg.Hint,
		d.propSvc,
		propertyService.CallBuilder().WithStoreOrTransaction(txq))
	if err != nil {
		return nil, fmt.Errorf("error getting parent entity: %w", err)
	}

	childEwp, err := getEntityInner(
		ctx,
		entMsg.Entity.Type, entMsg.Entity.GetByProps, entMsg.Hint,
		d.propSvc,
		propertyService.CallBuilder().WithStoreOrTransaction(txq))
	if err != nil {
		return nil, fmt.Errorf("error getting parent entity: %w", err)
	}

	err = txq.DeleteEntity(ctx, db.DeleteEntityParams{
		ID:        childEwp.Entity.ID,
		ProjectID: childEwp.Entity.ProjectID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err := d.store.Commit(tx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return nil, nil
}

// GetName returns the name of the strategy. Used for debugging
func (_ *delOriginatingEntityStrategy) GetName() string {
	return "delOriginatingEntityStrategy"
}
