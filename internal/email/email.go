// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package email provides the email utilities for minder
package email

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
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

var (
	// htmlTagRegex is a regex to match HTML tags
	htmlTagRegex = regexp.MustCompile(`<\/?[a-z][\s\S]*?>`)
	// htmlEntityRegex is a regex to match HTML entities
	htmlEntityRegex = regexp.MustCompile(`&[a-zA-Z0-9#]+;`)
	// htmlCommentRegex is a regex to match HTML comments
	htmlCommentRegex = regexp.MustCompile(`<!--[\s\S]*?-->`)
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

	// Validate the data source template for HTML injection attacks or empty fields
	err = validateDataSourceTemplate(&data)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error validating data source")
		return nil, fmt.Errorf("error validating data source: %w", err)
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

// isValidField checks if a string contains HTML tags, entities, or comments.
func isValidField(str string) error {
	if str == "" {
		return fmt.Errorf("string is empty")
	}

	// Check for HTML tags, entities, or comments
	if htmlTagRegex.MatchString(str) || htmlEntityRegex.MatchString(str) || htmlCommentRegex.MatchString(str) {
		return fmt.Errorf("string %s contains HTML tags, entities, or comments", str)
	}
	return nil
}

// validateDataSourceTemplate validates the template data source for HTML injection attacks
func validateDataSourceTemplate(s interface{}) error {
	// Get the reflect.Value of the pointer to the struct
	v := reflect.ValueOf(s).Elem()

	// Iterate over the fields of the struct
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		// Check if the field is settable and of kind string
		if field.CanSet() && field.Kind() == reflect.String {
			strVal := field.String()

			// Execute your function on the field value
			err := isValidField(strVal)
			if err != nil {
				return fmt.Errorf("field %s is empty or contains HTML injection - %s", v.Type().Field(i).Name, strVal)
			}
		}
	}
	return nil
}
