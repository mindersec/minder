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

package github

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/stacklok/minder/internal/entities/properties"
	properties2 "github.com/stacklok/minder/internal/providers/github/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// FetchProperty fetches a single property for the given entity
func (c *GitHub) FetchProperty(
	ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, key string,
) (*properties.Property, error) {
	if c.propertyFetchers == nil {
		return nil, errors.New("property fetchers not initialized")
	}

	fetcher := c.propertyFetchers.EntityPropertyFetcher(entType)

	// TODO: right now github only supports fetching by name, but we could add support for more
	// properties to e.g. get-repo-by-id if the upstream REST API supports that
	name, err := getByProps.GetProperty(properties.PropertyName).AsString()
	if err != nil {
		return nil, fmt.Errorf("name is not a string: %w", err)
	}

	wrapper := fetcher.WrapperForProperty(key)
	if wrapper == nil {
		return nil, fmt.Errorf("property %s not supported for entity %s", key, entType)
	}

	props, err := wrapper(ctx, c.client, name)
	if err != nil {
		return nil, fmt.Errorf("error fetching property %s for entity %s: %w", key, entType, err)
	}
	value, ok := props[key]
	if !ok {
		return nil, errors.New("requested property not found in result")
	}
	return properties.NewProperty(value)
}

// FetchAllProperties fetches all properties for the given entity
func (c *GitHub) FetchAllProperties(
	ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, cachedProps *properties.Properties,
) (*properties.Properties, error) {
	if c.propertyFetchers == nil {
		return nil, errors.New("property fetchers not initialized")
	}

	// TODO: right now github only supports fetching by name, but we could add support for more
	// properties to e.g. get-repo-by-id if the upstream REST API supports that
	name, err := getByProps.GetProperty(properties.PropertyName).AsString()
	if err != nil {
		return nil, fmt.Errorf("name is not a string: %w", err)
	}

	fetcher := c.propertyFetchers.EntityPropertyFetcher(entType)
	result := make(map[string]any)
	for _, wrapper := range fetcher.AllPropertyWrappers() {
		props, err := wrapper(ctx, c.client, name)
		if err != nil {
			return nil, fmt.Errorf("error fetching properties for entity %s: %w", entType, err)
		}

		for k, v := range props {
			result[k] = v
		}
	}

	upstreamProps, err := properties.NewProperties(result)
	if err != nil {
		return nil, err
	}

	operational := filterOperational(cachedProps, fetcher)
	return upstreamProps.Merge(operational), nil
}

func filterOperational(cachedProperties *properties.Properties, fetcher properties2.GhPropertyFetcher) *properties.Properties {
	operational := fetcher.OperationalProperties()
	if len(operational) == 0 {
		return cachedProperties
	}

	filter := func(key string, _ *properties.Property) bool {
		return slices.Contains(operational, key)
	}

	return cachedProperties.FilteredCopy(filter)
}
