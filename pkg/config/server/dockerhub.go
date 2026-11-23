// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// DockerHubConfig is the server-side configuration for the Docker Hub provider class
type DockerHubConfig struct {
	// WebhookSecrets is the configuration for the Docker Hub push webhook
	// secrets. Docker Hub does not sign webhook deliveries, so the secret is
	// used to derive a per-provider token that is carried in the webhook URL
	// and verified on each incoming request.
	WebhookSecrets `mapstructure:",squash"`
}
