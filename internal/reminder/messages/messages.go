// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package messages contains the messages used by the reminder service
package messages

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// EntityReminderEvent is an event that is published by the reminder service to trigger repo reconciliation
type EntityReminderEvent struct {
	// Project is the project that the event is relevant to
	Project uuid.UUID `json:"project"`
	// ProviderID is the provider of the repository
	ProviderID uuid.UUID `json:"provider"`
	// EntityID is the entity id of the repository to be reconciled
	EntityID uuid.UUID `json:"entity_id"`
}

// NewEntityReminderMessage creates a new repo reminder message
func NewEntityReminderMessage(providerId uuid.UUID, entityID uuid.UUID, projectID uuid.UUID) (*message.Message, error) {
	evt := &EntityReminderEvent{
		Project:    projectID,
		ProviderID: providerId,
		EntityID:   entityID,
	}

	evtStr, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling repo reminder event: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), evtStr)
	return msg, nil
}

// EntityReminderEventFromMessage creates a new repo reminder event from a message
func EntityReminderEventFromMessage(msg *message.Message) (*EntityReminderEvent, error) {
	var evt EntityReminderEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return nil, fmt.Errorf("error unmarshalling payload: %w", err)
	}

	return &evt, nil
}
