// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
