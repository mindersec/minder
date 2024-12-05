// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package message contains the message creation strategies
package message

import (
	"context"
	"fmt"

	watermill "github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
	"github.com/mindersec/minder/internal/reconcilers/messages"
)

type toMinderEntityStrategy struct{}

// NewToMinderEntity creates a new toMinderEntityStrategy.
func NewToMinderEntity() strategies.MessageCreateStrategy {
	return &toMinderEntityStrategy{}
}

func (_ *toMinderEntityStrategy) CreateMessage(_ context.Context, ewp *models.EntityWithProperties) (*watermill.Message, error) {
	if ewp == nil {
		return nil, fmt.Errorf("entity with properties is nil")
	}

	m := watermill.NewMessage(uuid.New().String(), nil)

	entEvent := messages.NewMinderEvent().
		WithProjectID(ewp.Entity.ProjectID).
		WithProviderID(ewp.Entity.ProviderID).
		WithEntityType(ewp.Entity.Type).
		WithEntityID(ewp.Entity.ID)

	err := entEvent.ToMessage(m)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to message: %w", err)
	}

	return m, nil
}

func (_ *toMinderEntityStrategy) GetName() string {
	return "toMinderv1Entity"
}
