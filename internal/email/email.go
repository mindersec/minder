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

// Package email provides the email utilities for minder
package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/events"
)

const (
	// TopicQueueInviteEmail is the topic for sending invite emails
	TopicQueueInviteEmail = "invite.email.event"
)

// Service is the email service interface
type Service interface {
	SendEmail(ctx context.Context, to, subject, bodyHTML, bodyText string) error
}

// MailEventPayload is the event payload for sending an invitation email
type MailEventPayload struct {
	Code           string `json:"code"`
	ProjectDisplay string `json:"project"`
	SponsorDisplay string `json:"sponsor"`
	Role           string `json:"role"`
	Email          string `json:"email"`
}

// MailManager is the email manager
type MailManager struct {
	client Service
}

// NewMailManager creates a new mail manager
func NewMailManager(client Service) *MailManager {
	return &MailManager{
		client: client,
	}
}

// Register implements the Consumer interface.
func (m *MailManager) Register(reg events.Registrar) {
	reg.Register(TopicQueueInviteEmail, m.handlerInviteEmail)
}

// handlerInviteEmail handles the invite email event
func (m *MailManager) handlerInviteEmail(msg *message.Message) error {
	var event MailEventPayload

	// Get the message context
	msgCtx := msg.Context()

	// Unmarshal the message payload
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("error unmarshalling invite email event: %w", err)
	}
	// Create the email subject and body
	// TODO: Use templates here
	subject := fmt.Sprintf("You have been invited to join %s", event.ProjectDisplay)
	bodyText := fmt.Sprintf("You have been invited to join %s as a %s by %s. Use code %s to accept the invitation.",
		event.ProjectDisplay, event.Role, event.SponsorDisplay, event.Code)
	bodyHtml := fmt.Sprintf("<p>%s</p>", bodyText)

	// Send the email
	return m.client.SendEmail(msgCtx, event.Email, subject, bodyHtml, bodyText)
}

// NewMessage creates a new message for sending an invitation email
func NewMessage(inviteeEmail, code, role, projectDisplay, sponsorDisplay string) (*message.Message, error) {
	// Generate a new message UUID
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error generating UUID: %w", err)
	}
	// Create the payload
	payload, err := json.Marshal(MailEventPayload{
		Code:           code,
		ProjectDisplay: projectDisplay,
		SponsorDisplay: sponsorDisplay,
		Role:           role,
		Email:          inviteeEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload for email event: %w", err)
	}
	// Create the message
	return message.NewMessage(id.String(), payload), nil
}
