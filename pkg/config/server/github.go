// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// GitHubConfig is the configuration for the GitHub OAuth providers
type GitHubConfig struct {
	OAuthClientConfig `mapstructure:",squash"`
}
