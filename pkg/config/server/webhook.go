// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"os"
	"strings"
)

// WebhookConfig is the configuration for our webhook capabilities
type WebhookConfig struct {
	// WebhookSecrets is the configuration for the webhook secrets.
	// This is embedded in the WebhookConfig so that the secrets can be
	// used in the WebhookConfig, as the GitHub provider needs for now.
	WebhookSecrets `mapstructure:",squash"`
	// ExternalWebhookURL is the URL that we will send our webhook to
	ExternalWebhookURL string `mapstructure:"external_webhook_url"`
	// ExternalPingURL is the URL that we will send our ping to
	ExternalPingURL string `mapstructure:"external_ping_url"`
}

// WebhookSecrets is the configuration for the webhook secrets. this is useful
// to import in whatever provider configuration that needs to use some webhook
// secrets.
type WebhookSecrets struct {
	// WebhookSecret is the secret that we will use to sign our webhook
	WebhookSecret string `mapstructure:"webhook_secret"`
	// WebhookSecretFile is the location of the file containing the webhook secret
	WebhookSecretFile string `mapstructure:"webhook_secret_file"`
	// PreviousWebhookSecretFile is a reference to a file that contains previous webhook secrets. This is used
	// in case we are rotating secrets and the external service is still using the old secret. These will not
	// be used when creating new webhooks.
	PreviousWebhookSecretFile string `mapstructure:"previous_webhook_secret_file"`
}

// GetPreviousWebhookSecrets retrieves the previous webhook secrets from a file specified in the WebhookConfig.
// It reads the contents of the file, splits the data by whitespace, and returns it as a slice of strings.
func (wc *WebhookSecrets) GetPreviousWebhookSecrets() ([]string, error) {
	data, err := os.ReadFile(wc.PreviousWebhookSecretFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read previous webhook secrets from file: %w", err)
	}

	// Split the data by whitespace and return it as a slice of strings
	secrets := strings.Fields(string(data))
	return secrets, nil
}

// GetWebhookSecret returns the GitHub App's webhook secret
func (wc *WebhookSecrets) GetWebhookSecret() (string, error) {
	return fileOrArg(wc.WebhookSecretFile, wc.WebhookSecret, "webhook secret")
}
