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

// Package awsses provides the email utilities for minder
package awsses

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/email"
	"github.com/stacklok/minder/internal/events"
)

const (
	// CharSet is the character set for the email
	CharSet = "UTF-8"
)

// awsSES is the AWS SES client
type awsSES struct {
	sender string
	svc    *sesv2.Client
}

// New creates a new AWS SES client
func New(ctx context.Context, sender, region string) (*awsSES, error) {
	// Load the AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	// Create an SES service client.
	return &awsSES{
		sender: sender,
		svc:    sesv2.NewFromConfig(cfg),
	}, nil
}

// Register implements the Consumer interface.
func (a *awsSES) Register(reg events.Registrar) {
	reg.Register(email.TopicQueueInviteEmail, func(msg *message.Message) error {
		var e email.MailEventPayload

		// Get the message context
		msgCtx := msg.Context()

		// Unmarshal the message payload
		if err := json.Unmarshal(msg.Payload, &e); err != nil {
			return fmt.Errorf("error unmarshalling invite email event: %w", err)
		}

		// Send the email
		return a.sendEmail(msgCtx, e.Address, e.Subject, e.BodyHTML, e.BodyText)
	})
}

// SendEmail sends an email using AWS SES
func (a *awsSES) sendEmail(ctx context.Context, to, subject, bodyHTML, bodyText string) error {
	zerolog.Ctx(ctx).Info().
		Str("invitee", to).
		Str("subject", subject).
		Msg("beginning to send email to invitee")

	// Assemble the email.
	input := &sesv2.SendEmailInput{
		// Set the email sender
		FromEmailAddress: aws.String(a.sender),
		// Set the email destination
		Destination: &types.Destination{
			CcAddresses: []string{},
			ToAddresses: []string{to},
		},
		// Set the email content
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Html: &types.Content{
						Charset: aws.String(CharSet),
						Data:    aws.String(bodyHTML),
					},
					Text: &types.Content{
						Charset: aws.String(CharSet),
						Data:    aws.String(bodyText),
					},
				},
				Subject: &types.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(subject),
				},
			},
		},
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	result, err := a.svc.SendEmail(ctx, input)

	// Display error messages if they occur.
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).
			Msg("error sending email")
		return err
	}

	// Log the message ID of the message sent to the user
	zerolog.Ctx(ctx).Info().
		Str("invitee", to).
		Str("emailMsgId", *result.MessageId).
		Msg("email sent successfully")

	return nil
}
