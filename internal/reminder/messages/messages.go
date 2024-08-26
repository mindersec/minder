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

// RepoReminderEvent is an event that is published by the reminder service to trigger repo reconciliation
type RepoReminderEvent struct {
	// Project is the project that the event is relevant to
	Project uuid.UUID `json:"project"`
	// ProviderID is the provider of the repository
	ProviderID uuid.UUID `json:"provider"`
	// RepositoryID is id of the repository to be reconciled
	RepositoryID int64 `json:"repository"`
}

// NewRepoReminderMessage creates a new repo reminder message
func NewRepoReminderMessage(providerId uuid.UUID, repoID int64, projectID uuid.UUID) (*message.Message, error) {
	evt := &RepoReminderEvent{
		Project:      projectID,
		ProviderID:   providerId,
		RepositoryID: repoID,
	}

	evtStr, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling repo reminder event: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), evtStr)
	return msg, nil
}

// RepoReminderEventFromMessage creates a new repo reminder event from a message
func RepoReminderEventFromMessage(msg *message.Message) (*RepoReminderEvent, error) {
	var evt RepoReminderEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return nil, fmt.Errorf("error unmarshalling payload: %w", err)
	}

	return &evt, nil
}
