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
	"github.com/stacklok/minder/internal/entities/properties"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	"github.com/stacklok/minder/internal/providers/manager"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
	childProps, err := properties.NewProperties(entMsg.Entity.GetByProps)
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

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

	parentEwp, err := getEntityInner(
		ctx,
		entMsg.Owner.Type, entMsg.Owner.GetByProps, entMsg.Hint,
		d.propSvc,
		propertyService.CallBuilder().WithStoreOrTransaction(txq))
	if err != nil {
		return nil, fmt.Errorf("error getting parent entity: %w", err)
	}

	prov, err := d.provMgr.InstantiateFromID(ctx, parentEwp.Entity.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("error getting provider: %w", err)
	}

	childEntName, err := prov.GetEntityName(entMsg.Entity.Type, childProps)
	if err != nil {
		return nil, fmt.Errorf("error getting child entity name: %w", err)
	}

	err = txq.DeleteEntityByName(ctx, db.DeleteEntityByNameParams{
		Name:      childEntName,
		ProjectID: parentEwp.Entity.ProjectID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	err = d.deleteLegacyEntity(ctx, entMsg.Entity.Type, parentEwp, childProps, txq)
	if err != nil {
		return nil, fmt.Errorf("error deleting legacy entity: %w", err)
	}

	if err := d.store.Commit(tx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return nil, nil
}

func (_ *delOriginatingEntityStrategy) deleteLegacyEntity(
	ctx context.Context,
	entType minderv1.Entity,
	parentEwp *models.EntityWithProperties,
	childProps *properties.Properties,
	t db.ExtendQuerier,
) error {
	if entType == minderv1.Entity_ENTITY_PULL_REQUESTS {
		err := t.DeletePullRequest(ctx, db.DeletePullRequestParams{
			RepositoryID: parentEwp.Entity.ID,
			PrNumber:     childProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
		})
		if err != nil {
			return fmt.Errorf("error deleting pull request: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported entity type: %v", entType)
	}

	return nil
}

// GetName returns the name of the strategy. Used for debugging
func (_ *delOriginatingEntityStrategy) GetName() string {
	return "delOriginatingEntityStrategy"
}
