//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func (ps *propertiesService) retrieveAllPropertiesForEntity(
	ctx context.Context, provider provifv1.Provider, entID uuid.UUID,
	lookupProperties *properties.Properties, entType minderv1.Entity,
	opts *ReadOptions, l zerolog.Logger,
) (*properties.Properties, error) {
	qtx := ps.getStoreOrTransaction(opts)

	var dbProps []db.Property
	if entID != uuid.Nil {
		l = l.With().Str("entityID", entID.String()).Logger()
		l.Debug().Msg("entity found, fetching properties")
		// fetch properties from db
		var err error
		dbProps, err = qtx.GetAllPropertiesForEntity(ctx, entID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	} else {
		l.Info().Msg("no entity found, skipping properties fetch")
	}

	// if exists and not expired, turn into our model
	var modelProps *properties.Properties
	if len(dbProps) > 0 {
		var err error
		modelProps, err = models.DbPropsToModel(dbProps)
		if err != nil {
			return nil, fmt.Errorf("failed to convert properties: %w", err)
		}
		if ps.areDatabasePropertiesValid(dbProps, opts) {
			l.Info().Msg("properties are valid, skipping provider fetch")
			return modelProps, nil
		}
	}

	// if not, fetch from provider
	l.Debug().Msg("properties are not valid, fetching from provider")
	refreshedProps, err := provider.FetchAllProperties(ctx, lookupProperties, entType, modelProps)
	if errors.Is(err, provifv1.ErrEntityNotFound) {
		return nil, fmt.Errorf("failed to fetch upstream properties: %w", ErrEntityNotFound)
	} else if err != nil {
		return nil, err
	}

	// if there was no entity, just return the properties as there is nothing to update. It's up to the caller
	// to decide what to do with the properties
	if entID == uuid.Nil {
		l.Debug().Msg("no entity found, returning properties without saving")
		return refreshedProps, nil
	}

	// save updated properties to db, thus making sure that the updatedAt are bumped
	err = ps.ReplaceAllProperties(ctx, entID, refreshedProps, opts.getPropertiesServiceCallOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to update properties: %w", err)
	}

	l.Debug().Msg("properties updated")
	return refreshedProps, nil
}

func getEntityIdByProperties(
	ctx context.Context, projectId uuid.UUID,
	providerID uuid.UUID,
	props *properties.Properties, entType minderv1.Entity,
	qtx db.ExtendQuerier,
) (uuid.UUID, error) {
	upstreamID := props.GetProperty(properties.PropertyUpstreamID)
	if upstreamID != nil {
		ent, getErr := getEntityIdByUpstreamID(ctx, projectId, providerID, upstreamID.GetString(), entType, qtx)
		if getErr == nil {
			return ent, nil
		} else if !errors.Is(getErr, ErrEntityNotFound) {
			return uuid.Nil, getErr
		}
		// on ErrEntityNot fall back to name if no upstream ID is provided
		// it might be that the entity was created with a name, but the upstream ID is not yet available
	}

	// Fall back to name if no upstream ID is provided
	name := props.GetProperty(properties.PropertyName)
	if name != nil {
		return getEntityIdByName(ctx, projectId, providerID, name.GetString(), entType, qtx)
	}

	// returning nil ID and nil error would make us just go to the provider. Slow, but we'd continue.
	return uuid.Nil, nil
}

func getEntityIdByName(
	ctx context.Context, projectId uuid.UUID,
	providerID uuid.UUID,
	name string, entType minderv1.Entity,
	qtx db.ExtendQuerier,
) (uuid.UUID, error) {
	ent, err := qtx.GetEntityByName(ctx, db.GetEntityByNameParams{
		ProjectID:  projectId,
		Name:       name,
		EntityType: entities.EntityTypeToDB(entType),
		ProviderID: providerID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrEntityNotFound
	} else if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get entity by name: %w", err)
	}

	return ent.ID, nil
}

func getEntityIdByUpstreamID(
	ctx context.Context, projectId uuid.UUID,
	providerID uuid.UUID,
	upstreamID string, entType minderv1.Entity,
	qtx db.ExtendQuerier,
) (uuid.UUID, error) {
	ents, err := qtx.GetTypedEntitiesByPropertyV1(
		ctx,
		entities.EntityTypeToDB(entType),
		properties.PropertyUpstreamID,
		upstreamID,
		db.GetTypedEntitiesOptions{
			ProjectID:  projectId,
			ProviderID: providerID,
		})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrEntityNotFound
	} else if err != nil {
		return uuid.Nil, fmt.Errorf("error fetching entities by property: %w", err)
	}

	if len(ents) > 1 {
		return uuid.Nil, ErrMultipleEntities
	} else if len(ents) == 1 {
		return ents[0].ID, nil
	}

	// no entity found
	return uuid.Nil, ErrEntityNotFound
}

func (ps *propertiesService) areDatabasePropertiesValid(
	dbProps []db.Property, opts *ReadOptions) bool {
	// if the all the properties are to be valid, neither must be older than
	// the cache timeout
	for _, prop := range dbProps {
		if !ps.isDatabasePropertyValid(prop, opts) {
			return false
		}
	}
	return true
}

func (ps *propertiesService) isDatabasePropertyValid(
	dbProp db.Property, opts *ReadOptions) bool {
	if ps.entityTimeout == bypassCacheTimeout {
		return false
	}
	return time.Since(dbProp.UpdatedAt) < ps.entityTimeout || opts.canTolerateStaleData()
}
