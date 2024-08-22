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

// Package reminderprocessor processes the incoming reminders
package reminderprocessor

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog/log"

	"github.com/stacklok/minder/internal/events"
	reconcilermessages "github.com/stacklok/minder/internal/reconcilers/messages"
	remindermessages "github.com/stacklok/minder/internal/reminder/messages"
)

// ReminderProcessor processes the incoming reminders
type ReminderProcessor struct {
	evt events.Interface
}

// NewReminderProcessor creates a new ReminderProcessor
func NewReminderProcessor(evt events.Interface) *ReminderProcessor {
	return &ReminderProcessor{evt: evt}
}

// Register implements the Consumer interface.
func (rp *ReminderProcessor) Register(r events.Registrar) {
	r.Register(events.TopicQueueRepoReminder, rp.reminderMessageHandler)
}

func (rp *ReminderProcessor) reminderMessageHandler(msg *message.Message) error {
	evt, err := remindermessages.RepoReminderEventFromMessage(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling reminder event: %w", err)
	}

	log.Info().Msgf("Received reminder event: %v", evt)

	repoReconcileMsg, err := reconcilermessages.NewRepoReconcilerMessage(evt.ProviderID, evt.RepositoryID, evt.Project)
	if err != nil {
		return fmt.Errorf("error creating repo reconcile event: %w", err)
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := rp.evt.Publish(events.TopicQueueReconcileRepoInit, repoReconcileMsg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}
	return nil
}
