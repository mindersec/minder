// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// PropValueV1 is the version string for v1 property values
const PropValueV1 = "v1"

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
	return propValueToDb(value, PropValueV1)
}

// PropValueFromDbV1 deserializes a property value from a JSON byte slice
func PropValueFromDbV1(rawValue json.RawMessage) (any, error) {
	return propValueFromDb(rawValue, PropValueV1)
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

// GetTypedEntitiesByPropertyV1 retrieves all entities with a property value
func (q *Queries) GetTypedEntitiesByPropertyV1(
	ctx context.Context, entType Entities, key string, value any, opts GetTypedEntitiesOptions,
) ([]EntityInstance, error) {
	jsonVal, err := PropValueToDbV1(value)
	if err != nil {
		return nil, err
	}

	return q.GetTypedEntitiesByProperty(ctx, GetTypedEntitiesByPropertyParams{
		EntityType: entType,
		ProjectID:  opts.ProjectID,
		ProviderID: opts.ProviderID,
		Key:        key,
		Value:      jsonVal,
	})
}

// PropMapFromJSON converts pre-fetched {"key": value} data from JSON_OBJECT_AGG
// into a map of property key to value, validating that each entry is v1.
func PropMapFromJSON(propsData json.RawMessage) (map[string]any, error) {
	var propData map[string]PropertyWrapper

	err := json.Unmarshal(propsData, &propData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling properties: %w", err)
	}

	propsMap := make(map[string]any, len(propData))
	for k, v := range propData {
		if v.Version != PropValueV1 {
			return nil, fmt.Errorf("property %q: %w", k, ErrBadPropVersion)
		}
		propsMap[k] = v.Value
	}
	return propsMap, nil
}
