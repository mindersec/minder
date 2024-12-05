// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package strategies contains the message creation strategies for entities and messages
package strategies

import (
	"context"

	watermill "github.com/ThreeDotsLabs/watermill/message"

	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/models"
)

// MessageCreateStrategy is the interface for creating messages
type MessageCreateStrategy interface {
	CreateMessage(
		ctx context.Context, ewp *models.EntityWithProperties,
	) (*watermill.Message, error)
	GetName() string
}

// GetEntityStrategy is the interface for getting entities
type GetEntityStrategy interface {
	GetEntity(
		ctx context.Context, entMsg *message.HandleEntityAndDoMessage,
	) (*models.EntityWithProperties, error)
	GetName() string
}
