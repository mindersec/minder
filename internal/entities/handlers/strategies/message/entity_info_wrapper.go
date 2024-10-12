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

// Package message contains the message creation strategies
package message

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	"github.com/stacklok/minder/internal/entities/models"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/providers/manager"
)

type toEntityInfoWrapper struct {
	store   db.Store
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
}

// NewToEntityInfoWrapper creates a new toEntityInfoWrapper.
func NewToEntityInfoWrapper(
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
) strategies.MessageCreateStrategy {
	return &toEntityInfoWrapper{
		store:   store,
		propSvc: propSvc,
		provMgr: provMgr,
	}
}

func (c *toEntityInfoWrapper) CreateMessage(
	ctx context.Context, ewp *models.EntityWithProperties,
) (*message.Message, error) {
	if ewp == nil {
		return nil, fmt.Errorf("entity with properties is nil")
	}

	pbEnt, err := c.propSvc.EntityWithPropertiesAsProto(ctx, ewp, c.provMgr)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	m := message.NewMessage(uuid.New().String(), nil)

	eiw := entities.NewEntityInfoWrapper().
		WithProjectID(ewp.Entity.ProjectID).
		WithProviderID(ewp.Entity.ProviderID).
		WithProtoMessage(ewp.Entity.Type, pbEnt).
		WithID(ewp.Entity.ID)

	err = eiw.ToMessage(m)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to message: %w", err)
	}

	return m, nil
}

func (_ *toEntityInfoWrapper) GetName() string {
	return "toEntityInfoWrapper"
}
