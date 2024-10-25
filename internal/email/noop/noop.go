// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package noop provides a noop email utilities for minder
package noop

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/email"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

type noop struct {
}

// New creates a new noop email service
func New() *noop {
	return &noop{}
}

// Register implements the Consumer interface.
func (_ *noop) Register(reg interfaces.Registrar) {
	reg.Register(email.TopicQueueInviteEmail, func(msg *message.Message) error {
		var e email.MailEventPayload

		// Get the message context
		msgCtx := msg.Context()

		// Unmarshal the message payload
		if err := json.Unmarshal(msg.Payload, &e); err != nil {
			return fmt.Errorf("error unmarshalling invite email event: %w", err)
		}

		// Log the email
		zerolog.Ctx(msgCtx).Info().
			Str("email", e.Address).
			Str("subject", e.Subject).
			Str("body_text", e.BodyText).
			Msg("Sending noop invite email")

		return nil
	})
}
