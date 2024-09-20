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

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	"github.com/stacklok/minder/internal/entities/models"
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
