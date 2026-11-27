// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package entity contains the entity creation strategies
package entity

import (
	"context"
	"fmt"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers/manager"
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

// refreshEntityByUpstreamProps looks up an entity by the upstream properties in the
// message and refreshes its properties from the provider within a transaction.
func refreshEntityByUpstreamProps(
	ctx context.Context,
	entMsg *message.HandleEntityAndDoMessage,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
) (*models.EntityWithProperties, error) {
	return db.WithTransaction(store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
		ewp, err := getEntityInner(
			ctx,
			entMsg.Entity.Type, entMsg.Entity.GetByProps, entMsg.Hint,
			propSvc, propertyService.CallBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error getting entity: %w", err)
		}

		err = propSvc.RetrieveAllPropertiesForEntity(
			ctx, ewp, provMgr,
			propertyService.ReadBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error fetching entity: %w", err)
		}
		return ewp, nil
	})
}
