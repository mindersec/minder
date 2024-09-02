// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package models contains domain models for entities
package models

import (
	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// EntityInstance represents an entity instance
type EntityInstance struct {
	ID         uuid.UUID
	Type       minderv1.Entity
	Name       string
	ProviderID uuid.UUID
	ProjectID  uuid.UUID
}

// EntityWithProperties represents an entity instance with properties
type EntityWithProperties struct {
	Entity     EntityInstance
	Properties *properties.Properties
}

// NewEntityWithProperties creates a new EntityWithProperties instance
func NewEntityWithProperties(dbEntity db.EntityInstance, props *properties.Properties) EntityWithProperties {
	return EntityWithProperties{
		Entity: EntityInstance{
			ID:         dbEntity.ID,
			Type:       entities.EntityTypeFromDB(dbEntity.EntityType),
			Name:       dbEntity.Name,
			ProviderID: dbEntity.ProviderID,
			ProjectID:  dbEntity.ProjectID,
		},
		Properties: props,
	}
}

// NewEntityWithPropertiesFromInstance creates a new EntityWithProperties instance from an existing entity instance
func NewEntityWithPropertiesFromInstance(entity EntityInstance, props *properties.Properties) EntityWithProperties {
	return EntityWithProperties{
		Entity:     entity,
		Properties: props,
	}
}

// EntityForProperties gives us the necessary fields to fetch properties
type EntityForProperties struct {
	*EntityWithProperties

	// Provider is the provider for the entity
	Provider provifv1.Provider
}

// NewEntityForProperties creates a new EntityForProperties instance
func NewEntityForProperties(
	dbEntity db.EntityInstance, props *properties.Properties, provider provifv1.Provider,
) *EntityForProperties {
	ewp := NewEntityWithProperties(dbEntity, props)
	return &EntityForProperties{
		EntityWithProperties: &ewp,
		Provider:             provider,
	}
}

// NewEntityForPropertiesFromInstance creates a new EntityForProperties instance from an existing entity instance
func NewEntityForPropertiesFromInstance(
	entity EntityInstance, props *properties.Properties, provider provifv1.Provider,
) *EntityForProperties {
	ewp := NewEntityWithPropertiesFromInstance(entity, props)
	return &EntityForProperties{
		EntityWithProperties: &ewp,
		Provider:             provider,
	}
}

// UpdateProperties updates the properties for the "entity for properties" instance
func (e *EntityForProperties) UpdateProperties(props *properties.Properties) {
	e.Properties = props
}
