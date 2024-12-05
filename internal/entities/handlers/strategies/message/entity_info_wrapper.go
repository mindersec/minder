// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package message contains the message creation strategies
package message

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers/manager"
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
