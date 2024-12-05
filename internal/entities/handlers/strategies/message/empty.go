// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package message contains the message creation strategies
package message

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	"github.com/mindersec/minder/internal/entities/models"
)

type createEmpty struct{}

// NewCreateEmpty creates a new createEmpty strategy
func NewCreateEmpty() strategies.MessageCreateStrategy {
	return &createEmpty{}
}

func (_ *createEmpty) CreateMessage(_ context.Context, _ *models.EntityWithProperties) (*message.Message, error) {
	return nil, nil
}

func (_ *createEmpty) GetName() string {
	return "empty"
}
