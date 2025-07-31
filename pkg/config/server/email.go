// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
)

// EmailConfig is the configuration for the email sending service
type EmailConfig struct {
	// MinderURLBase is the base URL to use for minder invite URLs, e.g. https://cloud.stacklok.com
	MinderURLBase string `mapstructure:"minder_url_base"`
	// AWSSES is the AWS SES configuration
	AWSSES AWSSES `mapstructure:"aws_ses"`
	// SendGrid is configuration for sending mail with Twilio's SendGrid, which has a free tier
	SendGrid SendGrid `mapstructure:"sendgrid"`
	// SMTP is the SMTP relay configuration
	SMTP SMTP `mapstructure:"smtp"`
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

// SMTP is the configuration for SMTP relay
type SMTP struct {
	// Sender is the email address of the sender
	Sender string `mapstructure:"sender"`
	// Host is the SMTP server hostname
	Host string `mapstructure:"host"`
	// Port is the SMTP server port
	Port int `mapstructure:"port" default:"0"`
	// Username is the SMTP username
	Username string `mapstructure:"username"`
	// PasswordFile is a file containing the SMTP password
	PasswordFile string `mapstructure:"password_file"`
}

// Validate checks the SMTP configuration for validity
func (cfg *SMTP) Validate() error {
	if cfg.Sender == "" {
		return fmt.Errorf("sender email address cannot be empty")
	}
	if cfg.Host == "" {
		return fmt.Errorf("SMTP host cannot be empty")
	}
	// 0 port means to use default (587 or 465)
	// If username/password are empty, we assume no authentication is required

	return nil
}
