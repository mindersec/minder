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

// Package noop provides a noop email utilities for minder
package noop

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/email"
	"github.com/stacklok/minder/internal/events"
)

type noop struct {
}

// New creates a new noop email service
func New() *noop {
	return &noop{}
}

// Register implements the Consumer interface.
func (_ *noop) Register(reg events.Registrar) {
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
