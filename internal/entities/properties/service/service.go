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
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/db"
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
	// EntityWithPropertiesAsProto calls the provider to convert the entity with properties to the appropriate proto message
	EntityWithPropertiesAsProto(
		ctx context.Context, ewp *models.EntityWithProperties, provMgr manager.ProviderManager,
	) (protoreflect.ProtoMessage, error)
	// EntityWithPropertiesByID Fetches an Entity by ID and Project in order to refresh the properties
	EntityWithPropertiesByID(
		ctx context.Context, entityID uuid.UUID, opts *CallOptions,
	) (*models.EntityWithProperties, error)
	// RetrieveAllProperties fetches all properties for an entity
	// given a project, provider, and identifying properties.
	// If the entity has properties in the database, it will return those
	// as long as they are not expired. Otherwise, it will fetch the properties
	// from the provider and update the database.
	RetrieveAllProperties(
		ctx context.Context, provider provifv1.Provider, projectId uuid.UUID,
		providerID uuid.UUID,
		lookupProperties *properties.Properties, entType minderv1.Entity,
		opts *ReadOptions,
	) (*properties.Properties, error)
	// RetrieveAllPropertiesForEntity fetches all properties for the given entity.
	// If the entity has properties in the database, it will return those
	// as long as they are not expired. Otherwise, it will fetch the properties
	// from the provider and update the database.
	// Note that this assumes an entity that already exists in Minder's database.
	RetrieveAllPropertiesForEntity(ctx context.Context, efp *models.EntityWithProperties,
		provMan manager.ProviderManager, opts *ReadOptions,
	) error
	// RetrieveProperty fetches a single property for the given entity given
	// a project, provider, and identifying properties.
	RetrieveProperty(
		ctx context.Context, provider provifv1.Provider, projectId uuid.UUID,
		providerID uuid.UUID,
		lookupProperties *properties.Properties, entType minderv1.Entity, key string,
		opts *ReadOptions,
	) (*properties.Property, error)
	// ReplaceAllProperties saves all properties for the given entity
	ReplaceAllProperties(
		ctx context.Context, entityID uuid.UUID, props *properties.Properties, opts *CallOptions,
	) error
	// SaveAllProperties saves all properties for the given entity
	SaveAllProperties(
		ctx context.Context, entityID uuid.UUID, props *properties.Properties, opts *CallOptions,
	) error
	// ReplaceProperty saves a single property for the given entity
	ReplaceProperty(
		ctx context.Context, entityID uuid.UUID, key string, prop *properties.Property, opts *CallOptions,
	) error
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

func (ps *propertiesService) RetrieveAllProperties(
	ctx context.Context, provider provifv1.Provider, projectId uuid.UUID,
	providerID uuid.UUID,
	lookupProperties *properties.Properties, entType minderv1.Entity,
	opts *ReadOptions,
) (*properties.Properties, error) {
	qtx := ps.getStoreOrTransaction(opts)
	l := zerolog.Ctx(ctx).With().
		Str("projectID", projectId.String()).
		Str("providerID", providerID.String()).
		Str("entityType", entType.String()).
		Logger()
	// fetch the entity first. If there's no entity, there's no properties, go straight to provider
	entID, err := getEntityIdByProperties(ctx, projectId, providerID, lookupProperties, entType, qtx)
	if err != nil && !errors.Is(err, ErrEntityNotFound) {
		return nil, fmt.Errorf("failed to get entity ID: %w", err)
	}

	return ps.retrieveAllPropertiesForEntity(ctx, provider, entID, lookupProperties, entType, opts, l)
}

func (ps *propertiesService) RetrieveAllPropertiesForEntity(
	ctx context.Context, efp *models.EntityWithProperties, provMan manager.ProviderManager,
	opts *ReadOptions,
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

	props, err := ps.retrieveAllPropertiesForEntity(ctx, propClient, efp.Entity.ID, efp.Properties, efp.Entity.Type, opts, l)
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
	opts *ReadOptions,
) (*properties.Property, error) {
	l := zerolog.Ctx(ctx).With().
		Str("projectID", projectId.String()).
		Str("providerID", providerID.String()).
		Str("entityType", entType.String()).
		Logger()

	// fetch the entity first. If there's no entity, there's no properties, go straight to provider
	entID, err := getEntityIdByProperties(ctx, projectId, providerID, lookupProperties, entType, ps.store)
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
	if ps.isDatabasePropertyValid(dbProp, opts) {
		l.Info().Msg("properties are valid, skipping provider fetch")
		return models.DbPropToModel(dbProp)
	}

	// if not, fetch from provider
	l.Debug().Msg("properties are not valid, fetching from provider")
	prop, err := provider.FetchProperty(ctx, lookupProperties, entType, key)
	if errors.Is(err, provifv1.ErrEntityNotFound) {
		return nil, fmt.Errorf("failed to fetch upstream property: %w", ErrEntityNotFound)
	} else if err != nil {
		return nil, err
	}

	return prop, nil
}

func (ps *propertiesService) ReplaceAllProperties(
	ctx context.Context, entityID uuid.UUID, props *properties.Properties,
	opts *CallOptions,
) error {
	qtx := ps.getStoreOrTransaction(opts)
	zerolog.Ctx(ctx).Debug().Str("entityID", entityID.String()).Msg("replacing all properties")

	err := qtx.DeleteAllPropertiesForEntity(ctx, entityID)
	if err != nil {
		return fmt.Errorf("failed to delete properties: %w", err)
	}

	return ps.SaveAllProperties(ctx, entityID, props, opts)
}

func (ps *propertiesService) SaveAllProperties(
	ctx context.Context, entityID uuid.UUID, props *properties.Properties,
	opts *CallOptions,
) error {
	qtx := ps.getStoreOrTransaction(opts)
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

func (ps *propertiesService) ReplaceProperty(
	ctx context.Context, entityID uuid.UUID, key string, prop *properties.Property,
	opts *CallOptions,
) error {
	qtx := ps.getStoreOrTransaction(opts)
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

func (ps *propertiesService) EntityWithPropertiesByID(
	ctx context.Context, entityID uuid.UUID,
	opts *CallOptions,
) (*models.EntityWithProperties, error) {
	q := ps.getStoreOrTransaction(opts)

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

// EntityWithPropertiesAsProto converts the entity with properties to a protobuf message
func (_ *propertiesService) EntityWithPropertiesAsProto(
	ctx context.Context, ewp *models.EntityWithProperties, provMgr manager.ProviderManager,
) (protoreflect.ProtoMessage, error) {
	prov, err := provMgr.InstantiateFromID(ctx, ewp.Entity.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("error instantiating provider %s: %w", ewp.Entity.ProviderID.String(), err)
	}

	converter, err := provifv1.As[provifv1.ProtoMessageConverter](prov)
	if err != nil {
		return nil, fmt.Errorf("provider %s doesn't implement ProtoMessageConverter: %w",
			ewp.Entity.ProviderID.String(), err)
	}

	return converter.PropertiesToProtoMessage(ewp.Entity.Type, ewp.Properties)
}
