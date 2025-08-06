//  SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package smtp provides the email utilities for minder using SMTP Relays
package smtp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"
	gomail "github.com/wneessen/go-mail"

	"github.com/mindersec/minder/internal/email"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// SMTP is the SMTP client
type SMTP struct {
	config     server.SMTP
	heloDomain string
	tlsConfig  *tls.Config // Currently only used for testing
}

// New creates a new SMTP client
func New(cfg server.SMTP) (*SMTP, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SMTP configuration: %w", err)
	}

	// Parse the sender email address to extract the domain for HELO
	senderAddr, err := mail.ParseAddress(cfg.Sender)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sender email address: %w", err)
	}
	domain := strings.Split(senderAddr.Address, "@")[1]

	return &SMTP{
		config:     cfg,
		heloDomain: domain,
	}, nil
}

// Register implements the Consumer interface.
func (s *SMTP) Register(reg interfaces.Registrar) {
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

// sendEmail sends an email using SMTP via go-mail library
func (s *SMTP) sendEmail(ctx context.Context, to, subject, bodyHTML, bodyText string) error {
	zerolog.Ctx(ctx).Info().
		Str("invitee", to).
		Str("subject", subject).
		Msg("beginning to send email to invitee")

	// Create a new message
	m := gomail.NewMsg()
	if err := m.From(s.config.Sender); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err := m.To(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	m.Subject(subject)
	m.SetBodyString(gomail.TypeTextPlain, bodyText)
	m.AddAlternativeString(gomail.TypeTextHTML, bodyHTML)

	opts := []gomail.Option{
		gomail.WithHELO(s.heloDomain),
	}

	if s.config.Username != "" {
		opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthPlain), gomail.WithUsername(s.config.Username))

		if s.config.PasswordFile != "" {
			passwordData, err := os.ReadFile(s.config.PasswordFile)
			if err != nil {
				return fmt.Errorf("failed to read SMTP password file: %w", err)
			}
			opts = append(opts, gomail.WithPassword(string(passwordData)))
		}
	}
	if s.config.Port != 0 {
		opts = append(opts, gomail.WithPort(s.config.Port))
	}
	if s.tlsConfig != nil {
		opts = append(opts, gomail.WithTLSConfig(s.tlsConfig))
	}

	// Create a new mail client
	client, err := gomail.NewClient(s.config.Host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}

	// Send the email
	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Log the successful email send
	zerolog.Ctx(ctx).Info().Str("invitee", to).Msg("email sent successfully")

	return nil
}
