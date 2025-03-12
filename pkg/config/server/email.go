// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// EmailConfig is the configuration for the email sending service
type EmailConfig struct {
	// MinderURLBase is the base URL to use for minder invite URLs, e.g. https://cloud.stacklok.com
	MinderURLBase string `mapstructure:"minder_url_base"`
	// AWSSES is the AWS SES configuration
	AWSSES AWSSES `mapstructure:"aws_ses"`
	// SendGrid is configuration for sending mail with Twilio's SendGrid, which has a free tier
	SendGrid SendGrid `mapstructure:"sendgrid"`
}

// AWSSES is the AWS SES configuration
type AWSSES struct {
	// Sender is the email address of the sender
	Sender string `mapstructure:"sender"`
	// Region is the AWS region to use for AWS SES
	Region string `mapstructure:"region"`
}

// SendGrid is the configuration for Twilio SendGrid
type SendGrid struct {
	// Sender is the email address of the sender
	Sender string `mapstructure:"sender"`
	// ApiKeyFile is a file containing the Twilio API key
	ApiKeyFile string `mapstructure:"api_key_file"`
}
