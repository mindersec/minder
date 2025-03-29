// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package entity contains the entity creation strategies
package entity

import (
	"context"
	"fmt"

	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
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

	lookupProperties := properties.NewProperties(entPropMap)

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
