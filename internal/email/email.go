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
	"github.com/rs/zerolog"
)

const (
	// TopicQueueInviteEmail is the topic for sending invite emails
	TopicQueueInviteEmail = "invite.email.event"
	// DefaultMinderTermsURL is the default terms URL for minder
	DefaultMinderTermsURL = "https://stacklok.com/stacklok-terms-of-service"
	// DefaultMinderPrivacyURL is the default privacy URL for minder
	DefaultMinderPrivacyURL = "https://stacklok.com/privacy-policy/"
)

// MailEventPayload is the event payload for sending an invitation email
type MailEventPayload struct {
	Address  string `json:"email"`
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
	BodyText string `json:"body_text"`
}

// NewMessage creates a new message for sending an invitation email
func NewMessage(
	ctx context.Context,
	inviteeEmail, inviteURL, minderURLBase, role, projectDisplay, sponsorDisplay string,
) (*message.Message, error) {
	// Generate a new message UUID
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error generating UUID: %w", err)
	}

	// Create the payload
	payload, err := json.Marshal(MailEventPayload{
		Address:  inviteeEmail,
		Subject:  getEmailSubject(projectDisplay),
		BodyHTML: getEmailBodyHTML(ctx, inviteURL, minderURLBase, sponsorDisplay, projectDisplay, role, inviteeEmail),
		BodyText: getEmailBodyText(inviteURL, sponsorDisplay, projectDisplay, role),
	})
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload for email event: %w", err)
	}

	// Create the message
	return message.NewMessage(id.String(), payload), nil
}

// getBodyHTML returns the HTML body for the email based on the message payload
func getEmailBodyHTML(ctx context.Context, inviteURL, minderURL, sponsor, project, role, inviteeEmail string) string {
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
		InvitationURL:    inviteURL,
		RecipientEmail:   inviteeEmail,
		MinderURL:        minderURL,
		TermsURL:         DefaultMinderTermsURL,
		PrivacyURL:       DefaultMinderPrivacyURL,
		SignInURL:        minderURL,
		RoleName:         role,
	}

	// TODO: Load the email template from elsewhere

	// Parse the template
	tmpl, err := templates.Parse(bodyHTML)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error parsing the HTML template for email invitations")
		// Default to the text body
		return getEmailBodyText(inviteURL, sponsor, project, role)
	}
	// Execute the template
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return ""
	}
	return b.String()
}

// getEmailBodyText returns the text body for the email based on the message payload
func getEmailBodyText(inviteURL, sponsor, project, role string) string {
	return fmt.Sprintf("You have been invited to join %s as a %s by %s. Visit %s to accept the invitation.",
		project, role, sponsor, inviteURL)
}

// getEmailSubject returns the subject for the email based on the message payload
func getEmailSubject(project string) string {
	return fmt.Sprintf("You have been invited to join the %s organisation in Minder", project)
}
