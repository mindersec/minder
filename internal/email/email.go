// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package email provides the email utilities for minder
package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/util"
)

// ErrValidationFailed is returned when the template data source fails validation
var ErrValidationFailed = errors.New("validation failed")

// NewErrValidationFailed creates a new error for failed validation
func NewErrValidationFailed(fieldName, fieldValue string) error {
	msg := fmt.Sprintf("field %s failed validation %s", fieldName, fieldValue)
	return fmt.Errorf("%w: %s", ErrValidationFailed, msg)
}

const (
	// TopicQueueInviteEmail is the topic for sending invite emails
	TopicQueueInviteEmail = "invite.email.event"
	// DefaultMinderTermsURL is the default terms URL for minder
	DefaultMinderTermsURL = "https://stacklok.com/stacklok-terms-of-service"
	// DefaultMinderPrivacyURL is the default privacy URL for minder
	DefaultMinderPrivacyURL = "https://stacklok.com/privacy-policy/"
	// BodyMaxLength is the maximum length of the email body
	BodyMaxLength = 10000
	// MaxFieldLength is the maximum length of a string field
	MaxFieldLength = 200
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
		return nil, err
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
	if err = data.Validate(); err != nil {
		return nil, err
	}
	// Create the HTML and text bodies
	strBodyHTML, err := data.GetEmailBodyHTML(ctx)
	if err != nil {
		return nil, err
	}
	strBodyText, err := data.GetEmailBodyText(ctx)
	if err != nil {
		return nil, err
	}
	strSubject, err := data.GetEmailSubject()
	if err != nil {
		return nil, err
	}
	// Create the payload
	payload, err := json.Marshal(MailEventPayload{
		Address:  inviteeEmail,
		Subject:  strSubject,
		BodyHTML: strBodyHTML,
		BodyText: strBodyText,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload for email event: %w", err)
	}

	// Create the message
	return message.NewMessage(id.String(), payload), nil
}

// GetEmailBodyHTML returns the HTML body for the email based on the message payload.
// If there is an error creating the HTML body, it will try to return the text body instead
func (b *bodyData) GetEmailBodyHTML(ctx context.Context) (string, error) {
	var builder strings.Builder
	bHTML := bodyHTML

	bodyHTMLTmpl, err := util.NewSafeHTMLTemplate(&bHTML, "body-invite-html")
	if err != nil {
		return "", err
	}
	err = bodyHTMLTmpl.Execute(ctx, &builder, b, BodyMaxLength)
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

// GetEmailBodyText returns the text body for the email based on the message payload
func (b *bodyData) GetEmailBodyText(ctx context.Context) (string, error) {
	var builder strings.Builder
	bText := bodyText

	bodyTextTmpl, err := util.NewSafeHTMLTemplate(&bText, "body-invite-text")
	if err != nil {
		return "", err
	}
	err = bodyTextTmpl.Execute(ctx, &builder, b, BodyMaxLength)
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

// GetEmailSubject returns the subject for the email based on the message payload
func (b *bodyData) GetEmailSubject() (string, error) {
	err := isValidField(b.OrganizationName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("You have been invited to join the %s organization in Minder", b.OrganizationName), nil
}

// isValidField checks if the string is empty or contains HTML or JavaScript injection
// If we detect any HTML or JavaScript injection, we want to return an error rather than escaping the content
func isValidField(str string) error {
	// Check string length
	if len(str) > MaxFieldLength {
		return fmt.Errorf("field value %s is more than %d characters", str, MaxFieldLength)
	}
	// Check for HTML content
	escapedHTML := template.HTMLEscapeString(str)
	if escapedHTML != str {
		return fmt.Errorf("string %s contains HTML injection", str)
	}

	// Check for JavaScript content separately
	escapedJS := template.JSEscaper(str)
	if escapedJS != str {
		return fmt.Errorf("string %s contains JavaScript injection", str)
	}

	return nil
}

// Validate validates the template data source for HTML injection attacks
func (b *bodyData) Validate() error {
	v := reflect.ValueOf(b).Elem()
	// Iterate over the fields of the struct
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		// Check if the field is settable and of kind and type string
		if !field.CanSet() || field.Kind() != reflect.String || field.Type() != reflect.TypeOf("") {
			return NewErrValidationFailed(v.Type().Field(i).Name, field.String())
		}
		err := isValidField(field.String())
		if err != nil {
			return NewErrValidationFailed(v.Type().Field(i).Name, field.String())
		}
	}
	return nil
}
