// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers/manager"
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
func (*delOriginatingEntityStrategy) GetName() string {
	return "delOriginatingEntityStrategy"
}
