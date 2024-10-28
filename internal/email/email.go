// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package email provides the email utilities for minder
package email

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/util"
)

const (
	// TopicQueueInviteEmail is the topic for sending invite emails
	TopicQueueInviteEmail = "invite.email.event"
	// DefaultMinderTermsURL is the default terms URL for minder
	DefaultMinderTermsURL = "https://stacklok.com/stacklok-terms-of-service"
	// DefaultMinderPrivacyURL is the default privacy URL for minder
	DefaultMinderPrivacyURL = "https://stacklok.com/privacy-policy/"
	// EmailBodyMaxLength is the maximum length of the email body
	EmailBodyMaxLength = 10000
)

// MailEventPayload is the event payload for sending an invitation email
type MailEventPayload struct {
	Address  string `json:"email"`
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
	BodyText string `json:"body_text"`
}

type bodyData struct {
	AdminName        string
	OrganizationName string
	InvitationURL    string
	RecipientEmail   string
	MinderURL        string
	TermsURL         string
	PrivacyURL       string
	SignInURL        string
	RoleName         string
	RoleVerb         string
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

	// Populate the template data source
	data := bodyData{
		AdminName:        sponsorDisplay,
		OrganizationName: projectDisplay,
		InvitationURL:    inviteURL,
		RecipientEmail:   inviteeEmail,
		MinderURL:        minderURLBase,
		TermsURL:         DefaultMinderTermsURL,
		PrivacyURL:       DefaultMinderPrivacyURL,
		SignInURL:        minderURLBase,
		RoleName:         role,
		RoleVerb:         authz.AllRolesVerbs[authz.Role(role)],
	}

	// Create the payload
	payload, err := json.Marshal(MailEventPayload{
		Address:  inviteeEmail,
		Subject:  getEmailSubject(projectDisplay),
		BodyHTML: getEmailBodyHTML(ctx, data),
		BodyText: getEmailBodyText(ctx, data),
	})
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload for email event: %w", err)
	}

	// Create the message
	return message.NewMessage(id.String(), payload), nil
}

// getHTMLBodyString returns the HTML body for the email based on the message payload.
// If there is an error creating the HTML body, it will try to return the text body instead
func getEmailBodyHTML(ctx context.Context, data bodyData) string {
	var b strings.Builder
	bHTML := bodyHTML

	bodyHTMLTmpl, err := util.NewSafeHTMLTemplate(&bHTML, "body-invite-html")
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error creating the HTML template for email invitations")
		return getEmailBodyText(ctx, data)
	}
	err = bodyHTMLTmpl.Execute(ctx, &b, data, EmailBodyMaxLength)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error executing the HTML template for email invitations")
		return getEmailBodyText(ctx, data)
	}
	return b.String()
}

// getTextBodyString returns the text body for the email based on the message payload
func getEmailBodyText(ctx context.Context, data bodyData) string {
	var b strings.Builder
	bText := bodyText

	bodyTextTmpl, err := util.NewSafeHTMLTemplate(&bText, "body-invite-text")
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error creating the text template for email invitations")
		return ""
	}
	err = bodyTextTmpl.Execute(ctx, &b, data, EmailBodyMaxLength)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error executing the text template for email invitations")
		return ""
	}
	return b.String()
}

// getEmailSubject returns the subject for the email based on the message payload
func getEmailSubject(project string) string {
	return fmt.Sprintf("You have been invited to join the %s organization in Minder", project)
}
