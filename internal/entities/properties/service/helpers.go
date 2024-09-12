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
	"slices"
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
	ctx context.Context,
	projectID uuid.UUID, providerID uuid.UUID,
	props *properties.Properties, entType minderv1.Entity,
	qtx db.ExtendQuerier,
) (uuid.UUID, error) {
	upstreamID := props.GetProperty(properties.PropertyUpstreamID)
	if upstreamID != nil {
		ent, getErr := getEntityIdByUpstreamID(ctx, projectID, providerID, upstreamID.GetString(), entType, qtx)
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
		return getEntityIdByName(ctx, projectID, providerID, name.GetString(), entType, qtx)
	}

	// returning nil ID and nil error would make us just go to the provider. Slow, but we'd continue.
	return uuid.Nil, nil
}

func getEntityIdByName(
	ctx context.Context, projectId uuid.UUID,
	providerId uuid.UUID,
	name string, entType minderv1.Entity,
	qtx db.ExtendQuerier,
) (uuid.UUID, error) {
	if projectId == uuid.Nil || providerId == uuid.Nil {
		return uuid.Nil, fmt.Errorf("projectID and providerID must be set")
	}

	ent, err := qtx.GetEntityByName(ctx, db.GetEntityByNameParams{
		ProjectID:  projectId,
		Name:       name,
		EntityType: entities.EntityTypeToDB(entType),
		ProviderID: providerId,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrEntityNotFound
	} else if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get entity by name: %w", err)
	}

	return ent.ID, nil
}

func getAllByUpstreamID(
	ctx context.Context,
	upstreamID string,
	entType minderv1.Entity,
	projectID uuid.UUID,
	providerID uuid.UUID,
	qtx db.ExtendQuerier,
) ([]db.EntityInstance, error) {
	l := zerolog.Ctx(ctx).With().Str("upstreamID", upstreamID).Logger()

	ents, err := qtx.GetTypedEntitiesByPropertyV1(
		ctx,
		entities.EntityTypeToDB(entType),
		properties.PropertyUpstreamID,
		upstreamID,
		db.GetTypedEntitiesOptions{
			ProjectID:  projectID,
			ProviderID: providerID,
		})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEntityNotFound
	} else if err != nil {
		return nil, fmt.Errorf("error fetching entities by property: %w", err)
	}

	if len(ents) == 0 {
		l.Debug().Msg("no entity found")
		return nil, ErrEntityNotFound
	}

	return ents, nil
}

func getEntityIdByUpstreamID(
	ctx context.Context,
	projectID uuid.UUID, providerID uuid.UUID,
	upstreamID string, entType minderv1.Entity,
	qtx db.ExtendQuerier,
) (uuid.UUID, error) {
	ents, err := getAllByUpstreamID(ctx, upstreamID, entType, projectID, providerID, qtx)
	if err != nil {
		return uuid.Nil, err
	}

	if len(ents) > 1 {
		return uuid.Nil, ErrMultipleEntities
	}

	return ents[0].ID, nil
}

func matchEntityByUpstreamID(
	ctx context.Context,
	upstreamID string, entType minderv1.Entity,
	hint *ByUpstreamIdHint,
	qtx db.ExtendQuerier,
) (*db.EntityInstance, error) {
	if hint.projectID == uuid.Nil && hint.providerID == uuid.Nil && !hint.ProviderImplements.Valid {
		return nil, fmt.Errorf("at least one of projectID, providerID or providerImplements must be set in hint")
	}

	ents, err := getAllByUpstreamID(ctx, upstreamID, entType, hint.projectID, hint.providerID, qtx)
	if err != nil {
		return nil, err
	}

	match, err := findMatchByUpstreamHint(ctx, ents, hint, qtx)
	if err != nil {
		l := zerolog.Ctx(ctx).With().Str("upstreamID", upstreamID).Logger()
		if errors.Is(err, ErrEntityNotFound) {
			l.Error().Msg("no entity matched")
		} else if errors.Is(err, ErrMultipleEntities) {
			l.Error().Msg("multiple entities matched")
		}
		return nil, err
	}

	return match, nil
}

func findMatchByUpstreamHint(
	ctx context.Context, ents []db.EntityInstance, hint *ByUpstreamIdHint, qtx db.ExtendQuerier,
) (*db.EntityInstance, error) {
	var match *db.EntityInstance
	for _, ent := range ents {
		var thisMatch *db.EntityInstance
		zerolog.Ctx(ctx).Debug().Msgf("matching entity %s", ent.ID.String())
		if dbEntMatchesUpstreamHint(ctx, ent, hint, qtx) {
			zerolog.Ctx(ctx).Debug().Msgf("entity %s matched by hint", ent.ID.String())
			thisMatch = &ent
		}

		if thisMatch != nil {
			if match != nil {
				zerolog.Ctx(ctx).Error().Msg("multiple entities matched")
				return nil, ErrMultipleEntities
			}
			match = thisMatch
		}
	}

	if match == nil {
		zerolog.Ctx(ctx).Debug().Msg("no entity matched")
		return nil, ErrEntityNotFound
	}

	return match, nil
}

func dbEntMatchesUpstreamHint(ctx context.Context, ent db.EntityInstance, hint *ByUpstreamIdHint, qtx db.ExtendQuerier) bool {
	logger := zerolog.Ctx(ctx)

	if hint.projectID != uuid.Nil {
		if ent.ProjectID != hint.projectID {
			logger.Debug().
				Str("projectID", ent.ProjectID.String()).
				Str("hintProjectID", hint.projectID.String()).
				Msg("project mismatch")
			return false
		}
	}

	if hint.providerID != uuid.Nil {
		if ent.ProviderID != hint.providerID {
			logger.Debug().
				Str("providerID", ent.ProviderID.String()).
				Str("hintProviderID", hint.providerID.String()).
				Msg("provider mismatch")
			return false
		}
	}

	if hint.ProviderImplements.Valid {
		dbProv, err := qtx.GetProviderByID(ctx, ent.ProviderID)
		if err != nil {
			logger.Error().
				Str("providerID", ent.ProviderID.String()).
				Err(err).
				Msg("error getting provider by ID")
			return false
		}

		if !slices.Contains(dbProv.Implements, hint.ProviderImplements.ProviderType) {
			logger.Debug().
				Str("ProviderID", ent.ProviderID.String()).
				Str("providerType", string(hint.ProviderImplements.ProviderType)).
				Msg("provider does not implement hint")
			return false
		}
	}

	return true
}

func propsMatchUpstreamHint(props *properties.Properties, hint ByUpstreamIdHint) bool {
	if hint.PropName == "" {
		return true
	}

	prop := props.GetProperty(hint.PropName)
	if prop == nil {
		return false
	}

	return prop.RawValue() == hint.PropValue
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
