//
// Copyright 2023 Stacklok, Inc.
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

// WebhookConfig is the configuration for our webhook capabilities
type WebhookConfig struct {
	// ExternalWebhookURL is the URL that we will send our webhook to
	ExternalWebhookURL string `mapstructure:"external_webhook_url"`
	// ExternalPingURL is the URL that we will send our ping to
	ExternalPingURL string `mapstructure:"external_ping_url"`
	// WebhookSecret is the secret that we will use to sign our webhook
	// TODO: Check if this is actually used and needed
	WebhookSecret string `mapstructure:"webhook_secret"`
}
