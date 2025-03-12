// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package sendgrid provides the email utilities for minder using SendGrid
package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/mindersec/minder/internal/email"
	config "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// SendGrid is the SendGrid client
type SendGrid struct {
	sender mail.Email
	client client
}

// Make an interface to support faking the client
type client interface {
	SendWithContext(ctx context.Context, msg *mail.SGMailV3) (*rest.Response, error)
}

// New creates a new SendGrid client
func New(cfg config.SendGrid) (*SendGrid, error) {
	if cfg.Sender == "" {
		return nil, fmt.Errorf("sender email address cannot be empty")
	}
	sender, err := mail.ParseEmail(cfg.Sender)
	if err != nil {
		return nil, fmt.Errorf("Incorrect sender email format: %s", err)
	}
	if cfg.ApiKeyFile == "" {
		return nil, fmt.Errorf("SendGrid API key cannot be empty")
	}
	apiKey, err := os.ReadFile(cfg.ApiKeyFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to read ApiKeyFile %s: %s", cfg.ApiKeyFile, err)
	}

	// Create a SendGrid client
	client := sendgrid.NewSendClient(string(apiKey))

	return &SendGrid{
		sender: *sender,
		client: client,
	}, nil
}

// Register implements the Consumer interface.
func (s *SendGrid) Register(reg interfaces.Registrar) {
	reg.Register(email.TopicQueueInviteEmail, func(msg *message.Message) error {
		var e email.MailEventPayload

		// Get the message context
		msgCtx := msg.Context()

		// Unmarshal the message payload
		if err := json.Unmarshal(msg.Payload, &e); err != nil {
			return fmt.Errorf("error unmarshalling invite email event: %w", err)
		}

		// Send the email
		return s.sendEmail(msgCtx, e.Address, e.Subject, e.BodyHTML, e.BodyText)
	})
}

// sendEmail sends an email using SendGrid
func (s *SendGrid) sendEmail(ctx context.Context, to, subject, bodyHTML, bodyText string) error {
	zerolog.Ctx(ctx).Info().
		Str("invitee", to).
		Str("subject", subject).
		Msg("beginning to send email to invitee")

	// TODO: figure out if we can get a friendly name as well as email address.
	toEmail := mail.NewEmail("", to)
	emailMsg := mail.NewSingleEmail(&s.sender, subject, toEmail, bodyText, bodyHTML)

	// Send the email
	response, err := s.client.SendWithContext(ctx, emailMsg)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error sending email")
		return err
	}

	// Check the response status code
	if response.StatusCode >= 400 {
		err := fmt.Errorf("SendGrid API error: status code %d, body: %s", response.StatusCode, response.Body)
		zerolog.Ctx(ctx).Error().Err(err).
			Int("statusCode", response.StatusCode).
			Str("body", response.Body).
			Msg("error sending email")
		return err
	}

	// Log the successful email send
	zerolog.Ctx(ctx).Info().Str("invitee", to).Msg("email sent successfully")

	return nil
}
