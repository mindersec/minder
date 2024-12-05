// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"

	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
)

type getEntityByUpstreamIDStrategy struct {
	propSvc propertyService.PropertiesService
}

// NewGetEntityByUpstreamIDStrategy creates a new getEntityByUpstreamIDStrategy.
func NewGetEntityByUpstreamIDStrategy(
	propSvc propertyService.PropertiesService,
) strategies.GetEntityStrategy {
	return &getEntityByUpstreamIDStrategy{
		propSvc: propSvc,
	}
}

// GetEntity gets an entity by its upstream ID.
func (g *getEntityByUpstreamIDStrategy) GetEntity(
	ctx context.Context, entMsg *message.HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	return getEntityInner(ctx,
		entMsg.Entity.Type, entMsg.Entity.GetByProps, entMsg.Hint,
		g.propSvc,
		propertyService.CallBuilder())
}

// GetName returns the name of the strategy. Used for debugging
func (_ *getEntityByUpstreamIDStrategy) GetName() string {
	return "getEntityByUpstreamID"
}
