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

// Package service provides a service to interact with properties of an entity
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
	"github.com/stacklok/minder/internal/providers/manager"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

var (
	// ErrEntityNotFound is returned when an entity is not found
	ErrEntityNotFound = errors.New("entity not found")
	// ErrMultipleEntities is returned when multiple entities are found
	ErrMultipleEntities = errors.New("multiple entities found")
	// ErrPropertyNotFound is returned when a property is not found
	ErrPropertyNotFound = errors.New("property not found")
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

const (
	// propertiesCacheTimeout is the default timeout for the cache of properties
	propertiesCacheTimeout = time.Duration(300) * time.Second
	// bypassCacheTimeout is a special value to bypass the cache timeout
	// it is not exported from the package and should only be used for testing
	bypassCacheTimeout = time.Duration(-1)
)

// PropertiesService is the interface for the properties service
type PropertiesService interface {
	// EntityWithProperties Fetches an Entity by ID and Project in order to refresh the properties
	EntityWithProperties(
		ctx context.Context, entityID uuid.UUID, qtx db.ExtendQuerier,
	) (*models.EntityWithProperties, error)
	// RetrieveAllProperties fetches all properties for the given entity
	RetrieveAllProperties(
		ctx context.Context, provider provifv1.Provider, projectId uuid.UUID,
		providerID uuid.UUID,
		lookupProperties *properties.Properties, entType minderv1.Entity,
	) (*properties.Properties, error)
	// RetrieveAllPropertiesForEntity fetches all properties for the given entity
	// for properties model. Note that properties will be updated in place.
	RetrieveAllPropertiesForEntity(ctx context.Context, efp *models.EntityWithProperties,
		provMan manager.ProviderManager,
	) error
	// RetrieveProperty fetches a single property for the given entity
	RetrieveProperty(
		ctx context.Context, provider provifv1.Provider, projectId uuid.UUID,
		providerID uuid.UUID,
		lookupProperties *properties.Properties, entType minderv1.Entity, key string,
	) (*properties.Property, error)
	// ReplaceAllProperties saves all properties for the given entity
	ReplaceAllProperties(ctx context.Context, entityID uuid.UUID, props *properties.Properties, qtx db.ExtendQuerier) error
	// SaveAllProperties saves all properties for the given entity
	SaveAllProperties(ctx context.Context, entityID uuid.UUID, props *properties.Properties, qtx db.ExtendQuerier) error
	// ReplaceProperty saves a single property for the given entity
	ReplaceProperty(ctx context.Context, entityID uuid.UUID, key string, prop *properties.Property, qtx db.ExtendQuerier) error
}

type propertiesServiceOption func(*propertiesService)

type propertiesService struct {
	store         db.ExtendQuerier
	entityTimeout time.Duration
}

// WithEntityTimeout sets the timeout for the cache of properties
func WithEntityTimeout(timeout time.Duration) propertiesServiceOption {
	return func(ps *propertiesService) {
		ps.entityTimeout = timeout
	}
}

// NewPropertiesService creates a new properties service
func NewPropertiesService(
	store db.ExtendQuerier,
	opts ...propertiesServiceOption,
) PropertiesService {
	ps := &propertiesService{
		store:         store,
		entityTimeout: propertiesCacheTimeout,
	}

	for _, opt := range opts {
		opt(ps)
	}

	return ps
}

// RetrieveAllProperties fetches a single property for the given entity
func (ps *propertiesService) RetrieveAllProperties(
	ctx context.Context, provider provifv1.Provider, projectId uuid.UUID,
	providerID uuid.UUID,
	lookupProperties *properties.Properties, entType minderv1.Entity,
) (*properties.Properties, error) {
	l := zerolog.Ctx(ctx).With().
		Str("projectID", projectId.String()).
		Str("providerID", providerID.String()).
		Str("entityType", entType.String()).
		Logger()

	// fetch the entity first. If there's no entity, there's no properties, go straight to provider
	entID, err := ps.getEntityIdByProperties(ctx, projectId, providerID, lookupProperties, entType)
	if err != nil && !errors.Is(err, ErrEntityNotFound) {
		return nil, fmt.Errorf("failed to get entity ID: %w", err)
	}

	var dbProps []db.Property
	if entID != uuid.Nil {
		l.Debug().Str("entityID", entID.String()).Msg("entity found, fetching properties")
		// fetch properties from db
		dbProps, err = ps.store.GetAllPropertiesForEntity(ctx, entID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	} else {
		l.Info().Msg("no entity found, skipping properties fetch")
	}

	// if exists and not expired, turn into our model
	var modelProps *properties.Properties
	if len(dbProps) > 0 {
		modelProps, err = models.DbPropsToModel(dbProps)
		if err != nil {
			return nil, fmt.Errorf("failed to convert properties: %w", err)
		}
		if ps.areDatabasePropertiesValid(dbProps) {
			l.Info().Msg("properties are valid, skipping provider fetch")
			return modelProps, nil
		}
	}

	// if not, fetch from provider
	l.Debug().Msg("properties are not valid, fetching from provider")
	refreshedProps, err := provider.FetchAllProperties(ctx, lookupProperties, entType, modelProps)
	if err != nil {
		return nil, err
	}

	// if there was no entity, just return the properties as there is nothing to update. It's up to the caller
	// to decide what to do with the properties
	if entID == uuid.Nil {
		l.Debug().Msg("no entity found, returning properties without saving")
		return refreshedProps, nil
	}

	// save updated properties to db, thus making sure that the updatedAt are bumped
	err = ps.ReplaceAllProperties(ctx, entID, refreshedProps, ps.store)
	if err != nil {
		return nil, fmt.Errorf("failed to update properties: %w", err)
	}

	l.Debug().Msg("properties updated")
	return refreshedProps, nil
}

// RetrieveAllPropertiesForEntity fetches a single property for the given an entity
// for properties model. Note that properties will be updated in place.
func (ps *propertiesService) RetrieveAllPropertiesForEntity(
	ctx context.Context, efp *models.EntityWithProperties, provMan manager.ProviderManager,
) error {
	l := zerolog.Ctx(ctx).With().
		Str("projectID", efp.Entity.ProjectID.String()).
		Str("providerID", efp.Entity.ProviderID.String()).
		Str("entityType", efp.Entity.Type.String()).
		Str("entityName", efp.Entity.Name).
		Str("entityID", efp.Entity.ID.String()).
		Logger()

	propClient, err := provMan.InstantiateFromID(ctx, efp.Entity.ProviderID)
	if err != nil {
		return fmt.Errorf("error instantiating provider: %w", err)
	}

	props, err := ps.RetrieveAllProperties(
		ctx,
		propClient,
		efp.Entity.ProjectID,
		efp.Entity.ProviderID,
		efp.Properties,
		efp.Entity.Type)
	if err != nil {
		return fmt.Errorf("error fetching properties for repository: %w", err)
	}

	efp.UpdateProperties(props)
	l.Debug().Msg("properties fetched and updated")
	return nil
}

// RetrieveProperty fetches a single property for the given entity
func (ps *propertiesService) RetrieveProperty(
	ctx context.Context, provider provifv1.Provider, projectId uuid.UUID,
	providerID uuid.UUID,
	lookupProperties *properties.Properties, entType minderv1.Entity, key string,
) (*properties.Property, error) {
	l := zerolog.Ctx(ctx).With().
		Str("projectID", projectId.String()).
		Str("providerID", providerID.String()).
		Str("entityType", entType.String()).
		Logger()

	// fetch the entity first. If there's no entity, there's no properties, go straight to provider
	entID, err := ps.getEntityIdByProperties(ctx, projectId, providerID, lookupProperties, entType)
	if err != nil && !errors.Is(err, ErrEntityNotFound) {
		return nil, err
	}

	// fetch properties from db
	var dbProp db.Property
	if entID != uuid.Nil {
		l.Debug().Str("entityID", entID.String()).Msg("entity found, fetching properties")
		dbProp, err = ps.store.GetProperty(ctx, db.GetPropertyParams{
			EntityID: entID,
			Key:      key,
		})
		if err != nil && !errors.Is(err, ErrPropertyNotFound) {
			return nil, err
		}
	} else {
		l.Info().Msg("no entity found, skipping properties fetch")
	}

	// if exists, turn into our model
	if ps.isDatabasePropertyValid(dbProp) {
		l.Info().Msg("properties are valid, skipping provider fetch")
		return models.DbPropToModel(dbProp)
	}

	// if not, fetch from provider
	l.Debug().Msg("properties are not valid, fetching from provider")
	prop, err := provider.FetchProperty(ctx, lookupProperties, entType, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch property: %w", err)
	}

	return prop, nil
}

func (ps *propertiesService) getEntityIdByProperties(
	ctx context.Context, projectId uuid.UUID,
	providerID uuid.UUID,
	props *properties.Properties, entType minderv1.Entity,
) (uuid.UUID, error) {
	upstreamID := props.GetProperty(properties.PropertyUpstreamID)
	if upstreamID != nil {
		ent, getErr := ps.getEntityIdByUpstreamID(ctx, projectId, providerID, upstreamID.GetString(), entType)
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
		return ps.getEntityIdByName(ctx, projectId, providerID, name.GetString(), entType)
	}

	// returning nil ID and nil error would make us just go to the provider. Slow, but we'd continue.
	return uuid.Nil, nil
}

func (ps *propertiesService) getEntityIdByName(
	ctx context.Context, projectId uuid.UUID,
	providerID uuid.UUID,
	name string, entType minderv1.Entity,
) (uuid.UUID, error) {
	ent, err := ps.store.GetEntityByName(ctx, db.GetEntityByNameParams{
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

func (ps *propertiesService) getEntityIdByUpstreamID(
	ctx context.Context, projectId uuid.UUID,
	providerID uuid.UUID,
	upstreamID string, entType minderv1.Entity,
) (uuid.UUID, error) {
	ents, err := ps.store.GetTypedEntitiesByPropertyV1(
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

func (ps *propertiesService) ReplaceAllProperties(
	ctx context.Context, entityID uuid.UUID, props *properties.Properties, qtx db.ExtendQuerier,
) error {
	zerolog.Ctx(ctx).Debug().Str("entityID", entityID.String()).Msg("replacing all properties")

	err := qtx.DeleteAllPropertiesForEntity(ctx, entityID)
	if err != nil {
		return fmt.Errorf("failed to delete properties: %w", err)
	}

	return ps.SaveAllProperties(ctx, entityID, props, qtx)
}

func (_ *propertiesService) SaveAllProperties(
	ctx context.Context, entityID uuid.UUID, props *properties.Properties, qtx db.ExtendQuerier,
) error {
	for key, prop := range props.Iterate() {
		_, err := qtx.UpsertPropertyValueV1(ctx, db.UpsertPropertyValueV1Params{
			EntityID: entityID,
			Key:      key,
			Value:    prop.RawValue(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (_ *propertiesService) ReplaceProperty(
	ctx context.Context, entityID uuid.UUID, key string, prop *properties.Property, qtx db.ExtendQuerier,
) error {
	if prop == nil {
		return qtx.DeleteProperty(ctx, db.DeletePropertyParams{
			EntityID: entityID,
			Key:      key,
		})
	}

	_, err := qtx.UpsertPropertyValueV1(ctx, db.UpsertPropertyValueV1Params{
		EntityID: entityID,
		Key:      key,
		Value:    prop.RawValue(),
	})
	return err
}

func (ps *propertiesService) EntityWithProperties(
	ctx context.Context, entityID uuid.UUID,
	qtx db.ExtendQuerier,
) (*models.EntityWithProperties, error) {
	// use the transaction if provided, otherwise use the store
	var q db.Querier
	if qtx != nil {
		q = qtx
	} else {
		q = ps.store
	}

	zerolog.Ctx(ctx).Debug().Str("entityID", entityID.String()).Msg("fetching entity with properties")
	ent, err := q.GetEntityByID(ctx, entityID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEntityNotFound
	} else if err != nil {
		return nil, fmt.Errorf("error getting entity: %w", err)
	}
	zerolog.Ctx(ctx).Debug().
		Str("projectID", ent.ProjectID.String()).
		Str("providerID", ent.ProviderID.String()).
		Str("entityType", string(ent.EntityType)).
		Str("entityName", ent.Name).
		Str("entityID", ent.ID.String()).
		Msg("entity found")

	dbProps, err := q.GetAllPropertiesForEntity(ctx, entityID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get properties for entity: %w", ErrEntityNotFound)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get properties for entity: %w", err)
	}

	props, err := models.DbPropsToModel(dbProps)
	if err != nil {
		return nil, fmt.Errorf("failed to convert properties to model: %w", err)
	}

	// temporary migration case - if we had an entity but no properties for it from
	// our live-on-demand migration case, we might not have a name. In this case, we
	// fill the name property from the entity name which is always there
	nameP := props.GetProperty(properties.PropertyName)
	if nameP == nil {
		err := props.SetKeyValue(properties.PropertyName, ent.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to set name property: %w", err)
		}
	}

	return models.NewEntityWithProperties(ent, props), nil
}

func (ps *propertiesService) areDatabasePropertiesValid(dbProps []db.Property) bool {
	// if the all the properties are to be valid, neither must be older than
	// the cache timeout
	for _, prop := range dbProps {
		if !ps.isDatabasePropertyValid(prop) {
			return false
		}
	}
	return true
}

func (ps *propertiesService) isDatabasePropertyValid(dbProp db.Property) bool {
	if ps.entityTimeout == bypassCacheTimeout {
		// this is mostly for testing
		return false
	}
	return time.Since(dbProp.UpdatedAt) < ps.entityTimeout
}
