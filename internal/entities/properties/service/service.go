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
	"time"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

const (
	// propertiesCacheTimeout is the default timeout for the cache of properties
	propertiesCacheTimeout = time.Duration(60) * time.Second
	// bypassCacheTimeout is a special value to bypass the cache timeout
	// it is not exported from the package and should only be used for testing
	bypassCacheTimeout = time.Duration(-1)
)

// PropertiesService is the interface for the properties service
type PropertiesService interface {
	// RetrieveAllProperties fetches all properties for the given entity
	RetrieveAllProperties(
		ctx context.Context, provider v1.Provider, projectId uuid.UUID,
		lookupProperties *properties.Properties, entType minderv1.Entity,
	) (*properties.Properties, error)
	// RetrieveProperty fetches a single property for the given entity
	RetrieveProperty(
		ctx context.Context, provider v1.Provider, projectId uuid.UUID,
		lookupProperties *properties.Properties, entType minderv1.Entity, key string,
	) (*properties.Property, error)
	// SaveAllProperties saves all properties for the given entity
	SaveAllProperties(ctx context.Context, entityID uuid.UUID, props *properties.Properties, qtx db.ExtendQuerier) error
	// SaveProperty saves a single property for the given entity
	SaveProperty(ctx context.Context, entityID uuid.UUID, key string, prop *properties.Property, qtx db.ExtendQuerier) error
}

type propertiesServiceOption func(*propertiesService)

type propertiesService struct {
	store         db.Store
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
	store db.Store,
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
	ctx context.Context, provider v1.Provider, projectId uuid.UUID,
	lookupProperties *properties.Properties, entType minderv1.Entity,
) (*properties.Properties, error) {
	// fetch the entity first. If there's no entity, there's no properties, go straight to provider
	entID, err := ps.getEntityIdByProperties(ctx, projectId, lookupProperties, entType)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var dbProps []db.Property
	if entID != uuid.Nil {
		// fetch properties from db
		dbProps, err = ps.store.GetAllPropertiesForEntity(ctx, entID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// if exists and not expired, turn into our model
	if len(dbProps) > 0 && ps.areDatabasePropertiesValid(dbProps) {
		// TODO: instead of a hard error, should we just re-fetch from provider?
		return dbPropsToModel(dbProps)
	}

	// if not, fetch from provider
	props, err := provider.FetchAllProperties(ctx, lookupProperties, entType, nil)
	if err != nil {
		return nil, err
	}

	return props, nil
}

// RetrieveProperty fetches a single property for the given entity
func (ps *propertiesService) RetrieveProperty(
	ctx context.Context, provider v1.Provider, projectId uuid.UUID,
	lookupProperties *properties.Properties, entType minderv1.Entity, key string,
) (*properties.Property, error) {
	// fetch the entity first. If there's no entity, there's no properties, go straight to provider
	entID, err := ps.getEntityIdByProperties(ctx, projectId, lookupProperties, entType)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// fetch properties from db
	var dbProp db.Property
	if entID != uuid.Nil {
		dbProp, err = ps.store.GetProperty(ctx, db.GetPropertyParams{
			EntityID: entID,
			Key:      key,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// if exists, turn into our model
	if ps.isDatabasePropertyValid(dbProp) {
		return dbPropToModel(dbProp)
	}

	// if not, fetch from provider
	prop, err := provider.FetchProperty(ctx, lookupProperties, entType, key)
	if err != nil {
		return nil, err
	}

	return prop, nil
}

func (ps *propertiesService) getEntityIdByProperties(
	ctx context.Context, projectId uuid.UUID,
	props *properties.Properties, entType minderv1.Entity,
) (uuid.UUID, error) {
	// TODO: Add more ways to look up a property, e.g. by the upstream ID
	name := props.GetProperty(properties.PropertyName)
	if name != nil {
		return ps.getEntityIdByName(ctx, projectId, name.GetString(), entType)
	}

	// returning nil ID and nil error would make us just go to the provider. Slow, but we'd continue.
	return uuid.Nil, nil
}

func (ps *propertiesService) getEntityIdByName(
	ctx context.Context, projectId uuid.UUID,
	name string, entType minderv1.Entity,
) (uuid.UUID, error) {
	ent, err := ps.store.GetEntityByName(ctx, db.GetEntityByNameParams{
		ProjectID:  projectId,
		Name:       name,
		EntityType: entities.EntityTypeToDB(entType),
	})
	if err != nil {
		return uuid.Nil, err
	}

	return ent.ID, nil
}

func (_ *propertiesService) SaveAllProperties(
	ctx context.Context, entityID uuid.UUID, props *properties.Properties, qtx db.ExtendQuerier,
) error {
	err := qtx.DeleteAllPropertiesForEntity(ctx, entityID)
	if err != nil {
		return err
	}

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

func (_ *propertiesService) SaveProperty(
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

func dbPropsToModel(dbProps []db.Property) (*properties.Properties, error) {
	propMap := make(map[string]any)

	// TODO: should we change the property API to include a Set
	// and rather move the construction from a map to a separate method?
	// this double iteration is not ideal
	for _, prop := range dbProps {
		anyVal, err := db.PropValueFromDbV1(prop.Value)
		if err != nil {
			return nil, err
		}
		propMap[prop.Key] = anyVal
	}

	return properties.NewProperties(propMap)
}

func dbPropToModel(dbProp db.Property) (*properties.Property, error) {
	anyVal, err := db.PropValueFromDbV1(dbProp.Value)
	if err != nil {
		return nil, err
	}

	return properties.NewProperty(anyVal)
}
