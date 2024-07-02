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
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/docker/cli/templates"
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
	Address  string `json:"email"`
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
	BodyText string `json:"body_text"`
}

// MailEventHandler is the email event handler
type MailEventHandler struct {
	client Service
}

// NewMailEventHandler creates a new mail event handler
func NewMailEventHandler(client Service) *MailEventHandler {
	return &MailEventHandler{
		client: client,
	}
}

// Register implements the Consumer interface.
func (m *MailEventHandler) Register(reg events.Registrar) {
	reg.Register(TopicQueueInviteEmail, m.handlerInviteEmail)
}

// handlerInviteEmail handles the invite email event
func (m *MailEventHandler) handlerInviteEmail(msg *message.Message) error {
	var e MailEventPayload

	// Get the message context
	msgCtx := msg.Context()

	// Unmarshal the message payload
	if err := json.Unmarshal(msg.Payload, &e); err != nil {
		return fmt.Errorf("error unmarshalling invite email event: %w", err)
	}

	// Send the email
	return m.client.SendEmail(msgCtx, e.Address, e.Subject, e.BodyHTML, e.BodyText)
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
		Address:  inviteeEmail,
		Subject:  getEmailSubject(projectDisplay),
		BodyHTML: getEmailBodyHTML(code, sponsorDisplay, projectDisplay, role, inviteeEmail),
		BodyText: getEmailBodyText(code, sponsorDisplay, projectDisplay, role),
	})
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload for email event: %w", err)
	}
	// Create the message
	return message.NewMessage(id.String(), payload), nil
}

// getBodyHTML returns the HTML body for the email based on the message payload
func getEmailBodyHTML(code, sponsor, project, role, inviteeEmail string) string {
	data := struct {
		AdminName        string
		OrganizationName string
		InvitationURL    string
		RecipientEmail   string
		MinderURL        string
		TermsURL         string
		PrivacyURL       string
		SignInURL        string
		RoleName         string
	}{
		AdminName:        sponsor,
		OrganizationName: project,
		// TODO: Determine the correct environment for the invite URL and the rest of the URLs
		InvitationURL:  fmt.Sprintf("https://cloud.minder.com/join/%s", code),
		RecipientEmail: inviteeEmail,
		MinderURL:      "https://cloud.minder.com",
		TermsURL:       "https://cloud.minder.com/terms",
		PrivacyURL:     "https://cloud.minder.com/privacy",
		SignInURL:      "https://cloud.minder.com",
		RoleName:       role,
	}

	// TODO: Load the email template from elsewhere

	// Parse the template
	tmpl, err := templates.Parse(bodyHTML)
	if err != nil {
		// TODO: Log the error
		// Default to the text body
		return getEmailBodyText(code, sponsor, project, role)
	}
	// Execute the template
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return ""
	}
	return b.String()
}

// getEmailBodyText returns the text body for the email based on the message payload
func getEmailBodyText(code, sponsor, project, role string) string {
	return fmt.Sprintf("You have been invited to join %s as a %s by %s. Use code %s to accept the invitation.",
		project, role, sponsor, code)
}

// getEmailSubject returns the subject for the email based on the message payload
func getEmailSubject(project string) string {
	return fmt.Sprintf("You have been invited to join the %s organisation in Minder", project)
}
