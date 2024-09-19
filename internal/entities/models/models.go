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
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// EntityInstance represents an entity instance
type EntityInstance struct {
	ID         uuid.UUID
	Type       minderv1.Entity
	Name       string
	ProviderID uuid.UUID
	ProjectID  uuid.UUID
}

// String implements fmt.Stringer for debugging purposes
func (ei EntityInstance) String() string {
	return fmt.Sprintf("[%s]<%s>: %s / %s (%s)", ei.ProviderID, ei.Type, ei.ProjectID, ei.ID, ei.Name)
}

// EntityWithProperties represents an entity instance with properties
type EntityWithProperties struct {
	Entity     EntityInstance
	Properties *properties.Properties
}

// NewEntityWithProperties creates a new EntityWithProperties instance
func NewEntityWithProperties(dbEntity db.EntityInstance, props *properties.Properties) *EntityWithProperties {
	return &EntityWithProperties{
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
func NewEntityWithPropertiesFromInstance(entity EntityInstance, props *properties.Properties) *EntityWithProperties {
	return &EntityWithProperties{
		Entity:     entity,
		Properties: props,
	}
}

// String implements fmt.Stringer for debugging purposes
func (ewp EntityWithProperties) String() string {
	return fmt.Sprintf("ENTITY %s:\n%s", ewp.Entity, ewp.Properties)
}

// DbPropsToModel converts a slice of db.Property to a properties.Properties instance.
func DbPropsToModel(dbProps []db.Property) (*properties.Properties, error) {
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

// DbPropToModel converts a single db.Property to a properties.Property instance.
func DbPropToModel(dbProp db.Property) (*properties.Property, error) {
	anyVal, err := db.PropValueFromDbV1(dbProp.Value)
	if err != nil {
		return nil, err
	}

	return properties.NewProperty(anyVal)
}

// UpdateProperties updates the properties for the "entity for properties" instance
func (e *EntityWithProperties) UpdateProperties(props *properties.Properties) {
	e.Properties = props
}
