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

package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

const propValueV1 = "v1"

// ErrBadPropVersion is returned when a property has an unexpected version
var ErrBadPropVersion = errors.New("unexpected property version")

// PropertyWrapper is a wrapper around a property value that includes a version to serialize as JSON
type PropertyWrapper struct {
	Version string `json:"version"`
	Value   any    `json:"value"`
}

func propValueFromDb(rawValue json.RawMessage, expVersion string) (any, error) {
	var prop PropertyWrapper
	err := json.Unmarshal(rawValue, &prop)
	if err != nil {
		return nil, err
	}

	if prop.Version != expVersion {
		return nil, ErrBadPropVersion
	}

	return prop.Value, nil
}

func propValueToDb(value any, version string) (json.RawMessage, error) {
	prop := PropertyWrapper{
		Version: version,
		Value:   value,
	}

	rawValue, err := json.Marshal(prop)
	if err != nil {
		return nil, err
	}

	return rawValue, nil
}

// PropValueToDbV1 serializes a property value to a JSON byte slice
func PropValueToDbV1(value any) (json.RawMessage, error) {
	return propValueToDb(value, propValueV1)
}

// PropValueFromDbV1 deserializes a property value from a JSON byte slice
func PropValueFromDbV1(rawValue json.RawMessage) (any, error) {
	return propValueFromDb(rawValue, propValueV1)
}

// UpsertPropertyValueV1Params is the input parameter for the UpsertProperty query
type UpsertPropertyValueV1Params struct {
	EntityID uuid.UUID `json:"entity_id"`
	Key      string    `json:"key"`
	Value    any       `json:"value"`
}

// UpsertPropertyValueV1 upserts a property value for an entity
func (q *Queries) UpsertPropertyValueV1(ctx context.Context, params UpsertPropertyValueV1Params) (Property, error) {
	jsonVal, err := PropValueToDbV1(params.Value)
	if err != nil {
		return Property{}, err
	}
	dbParams := UpsertPropertyParams{
		EntityID: params.EntityID,
		Key:      params.Key,
		Value:    jsonVal,
	}
	return q.UpsertProperty(ctx, dbParams)
}

// PropertyValueV1 is a property value for an entity
type PropertyValueV1 struct {
	ID        uuid.UUID `json:"id"`
	EntityID  uuid.UUID `json:"entity_id"`
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetPropertyValueV1 retrieves a property value for an entity
func (q *Queries) GetPropertyValueV1(ctx context.Context, entityID uuid.UUID, key string) (PropertyValueV1, error) {
	dbProp, err := q.GetProperty(ctx, GetPropertyParams{
		EntityID: entityID,
		Key:      key,
	})
	if err != nil {
		return PropertyValueV1{}, err
	}

	value, err := PropValueFromDbV1(dbProp.Value)
	if err != nil {
		return PropertyValueV1{}, err
	}

	return PropertyValueV1{
		ID:        dbProp.ID,
		EntityID:  dbProp.EntityID,
		Key:       dbProp.Key,
		Value:     value,
		UpdatedAt: dbProp.UpdatedAt,
	}, nil
}

// GetAllPropertyValuesV1 retrieves all property values for an entity
func (q *Queries) GetAllPropertyValuesV1(ctx context.Context, entityID uuid.UUID) ([]PropertyValueV1, error) {
	dbProps, err := q.GetAllPropertiesForEntity(ctx, entityID)
	if err != nil {
		return nil, err
	}

	props := make([]PropertyValueV1, len(dbProps))
	for i, dbProp := range dbProps {
		value, err := PropValueFromDbV1(dbProp.Value)
		if err != nil {
			return nil, err
		}

		props[i] = PropertyValueV1{
			ID:        dbProp.ID,
			EntityID:  dbProp.EntityID,
			Key:       dbProp.Key,
			Value:     value,
			UpdatedAt: dbProp.UpdatedAt,
		}
	}

	return props, nil
}
