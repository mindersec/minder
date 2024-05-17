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

import (
	"fmt"
)

// OAuthEndpoint is the configuration for the OAuth endpoint
// Only used for testing
type OAuthEndpoint struct {
	// TokenURL is the OAuth token URL
	TokenURL string `mapstructure:"token_url"`
}

// OAuthClientConfig is the configuration for the OAuth client
type OAuthClientConfig struct {
	// ClientID is the OAuth client ID
	ClientID string `mapstructure:"client_id"`
	// ClientIDFile is the location of the file containing the OAuth client ID
	ClientIDFile string `mapstructure:"client_id_file"`
	// ClientSecret is the OAuth client secret
	ClientSecret string `mapstructure:"client_secret"`
	// ClientSecretFile is the location of the file containing the OAuth client secret
	ClientSecretFile string `mapstructure:"client_secret_file"`
	// RedirectURI is the OAuth redirect URI
	RedirectURI string `mapstructure:"redirect_uri"`
	// Endpoint is the OAuth endpoint. Currently only used for testing
	Endpoint *OAuthEndpoint `mapstructure:"endpoint"`
}

// GetClientID returns the OAuth client ID from either the file or the argument
func (cfg *OAuthClientConfig) GetClientID() (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("OAuthClientConfig is nil")
	}
	return fileOrArg(cfg.ClientIDFile, cfg.ClientID, "client ID")
}

// GetClientSecret returns the OAuth client secret from either the file or the argument
func (cfg *OAuthClientConfig) GetClientSecret() (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("OAuthClientConfig is nil")
	}
	return fileOrArg(cfg.ClientSecretFile, cfg.ClientSecret, "client secret")
}
