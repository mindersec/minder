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

	watermill "github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/reconcilers/messages"
)

type toMinderEntityStrategy struct{}

// NewToMinderEntity creates a new toMinderEntityStrategy.
func NewToMinderEntity() strategies.MessageCreateStrategy {
	return &toMinderEntityStrategy{}
}

func (_ *toMinderEntityStrategy) CreateMessage(_ context.Context, ewp *models.EntityWithProperties) (*watermill.Message, error) {
	m := watermill.NewMessage(uuid.New().String(), nil)

	entEvent := messages.NewMinderEvent().
		WithProjectID(ewp.Entity.ProjectID).
		WithProviderID(ewp.Entity.ProviderID).
		WithEntityType(ewp.Entity.Type.String()).
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
