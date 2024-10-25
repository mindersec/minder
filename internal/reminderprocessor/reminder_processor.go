// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package reminderprocessor processes the incoming reminders
package reminderprocessor

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog/log"

	"github.com/mindersec/minder/internal/events"
	reconcilermessages "github.com/mindersec/minder/internal/reconcilers/messages"
	remindermessages "github.com/mindersec/minder/internal/reminder/messages"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// ReminderProcessor processes the incoming reminders
type ReminderProcessor struct {
	evt interfaces.Interface
}

// NewReminderProcessor creates a new ReminderProcessor
func NewReminderProcessor(evt interfaces.Interface) *ReminderProcessor {
	return &ReminderProcessor{evt: evt}
}

// Register implements the Consumer interface.
func (rp *ReminderProcessor) Register(r interfaces.Registrar) {
	r.Register(events.TopicQueueRepoReminder, rp.reminderMessageHandler)
}

func (rp *ReminderProcessor) reminderMessageHandler(msg *message.Message) error {
	evt, err := remindermessages.EntityReminderEventFromMessage(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling reminder event: %w", err)
	}

	log.Info().Msgf("Received reminder event: %v", evt)

	repoReconcileMsg, err := reconcilermessages.NewRepoReconcilerMessage(evt.ProviderID, evt.EntityID, evt.Project)
	if err != nil {
		return fmt.Errorf("error creating repo reconcile event: %w", err)
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := rp.evt.Publish(events.TopicQueueReconcileRepoInit, repoReconcileMsg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}
	return nil
}
