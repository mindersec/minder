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

	"github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	"github.com/stacklok/minder/internal/entities/models"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
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
