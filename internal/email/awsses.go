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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/rs/zerolog"
)

const (
	// CharSet is the character set for the email
	CharSet = "UTF-8"
	// DefaultAWSRegion is the default AWS region
	DefaultAWSRegion = "us-east-1"
	// DefaultSender is the default sender email address
	DefaultSender = "noreply@stacklok.com"
)

// AWSSES is the AWS SES client
type AWSSES struct {
	sender string
	svc    *ses.SES
}

// NewAWSSES creates a new AWS SES client
func NewAWSSES(sender, region string) (*AWSSES, error) {
	// Set the sender and region in case they are not provided.
	if sender == "" {
		sender = DefaultSender
	}
	if region == "" {
		region = DefaultAWSRegion
	}

	// Create a new session.
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, err
	}

	// Create an SES service client.
	return &AWSSES{
		sender: sender,
		svc:    ses.New(sess),
	}, nil
}

// SendEmail sends an email using AWS SES
func (a *AWSSES) SendEmail(ctx context.Context, to, subject, bodyHTML, bodyText string) error {
	zerolog.Ctx(ctx).Info().
		Str("invitee", to).
		Str("subject", subject).
		Msg("beginning to send email to invitee")

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(to),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(bodyHTML),
				},
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(bodyText),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(a.sender),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	result, err := a.svc.SendEmail(input)

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
