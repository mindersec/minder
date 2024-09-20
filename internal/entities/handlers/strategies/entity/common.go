// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package entity contains the entity creation strategies
package entity

import (
	"context"
	"fmt"

	"github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func getEntityInner(
	ctx context.Context,
	entType minderv1.Entity,
	entPropMap map[string]any,
	hint message.EntityHint,
	propSvc propertyService.PropertiesService,
	getEntityOpts *propertyService.CallOptions,
) (*models.EntityWithProperties, error) {
	svcHint := propertyService.ByUpstreamHint{}
	if hint.ProviderImplementsHint != "" {
		svcHint.ProviderImplements.Valid = true
		if err := svcHint.ProviderImplements.ProviderType.Scan(hint.ProviderImplementsHint); err != nil {
			return nil, fmt.Errorf("error scanning provider type: %w", err)
		}
	}
	if hint.ProviderClassHint != "" {
		svcHint.ProviderClass.Valid = true
		if err := svcHint.ProviderClass.ProviderClass.Scan(hint.ProviderClassHint); err != nil {
			return nil, fmt.Errorf("error scanning provider class: %w", err)
		}
	}

	lookupProperties, err := properties.NewProperties(entPropMap)
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

	ewp, err := propSvc.EntityWithPropertiesByUpstreamHint(
		ctx,
		entType,
		lookupProperties,
		svcHint,
		getEntityOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("error searching entity by ID: %w", err)
	}

	return ewp, nil
}
