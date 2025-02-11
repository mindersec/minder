// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// GitLabConfig is the configuration for the GitLab OAuth providers
type GitLabConfig struct {
	OAuthClientConfig `mapstructure:",squash"`

	// WebhookSecrets is the configuration for the GitLab webhook secrets
	// setup and verification. This is used to verify incoming webhook requests
	// from GitLab, as well as to generate the webhook URL for GitLab to send
	// events to.
	WebhookSecrets `mapstructure:",squash"`

	// Scopes is the list of scopes to request from the GitLab OAuth provider
	Scopes []string `mapstructure:"scopes"`
}
