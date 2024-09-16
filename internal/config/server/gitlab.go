//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
