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

	"github.com/spf13/viper"
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

// FallbackOAuthClientConfigValues reads the OAuth client configuration values directly via viper
// this is a temporary hack until we migrate all the configuration to be read from the per-provider
// sections
func FallbackOAuthClientConfigValues(provider string, cfg *OAuthClientConfig) {
	// we read the values one-by-one instead of just getting the top-level key to allow
	// for environment variables to be set per-variable
	fallbackClientID := viper.GetString(fmt.Sprintf("%s.client_id", provider))
	if fallbackClientID != "" {
		cfg.ClientID = fallbackClientID
	}

	fallbackClientIDFile := viper.GetString(fmt.Sprintf("%s.client_id_file", provider))
	if fallbackClientIDFile != "" {
		cfg.ClientIDFile = fallbackClientIDFile
	}

	fallbackClientSecret := viper.GetString(fmt.Sprintf("%s.client_secret", provider))
	if fallbackClientSecret != "" {
		cfg.ClientSecret = fallbackClientSecret
	}

	fallbackClientSecretFile := viper.GetString(fmt.Sprintf("%s.client_secret_file", provider))
	if fallbackClientSecretFile != "" {
		cfg.ClientSecretFile = fallbackClientSecretFile
	}

	fallbackRedirectURI := viper.GetString(fmt.Sprintf("%s.redirect_uri", provider))
	if fallbackRedirectURI != "" {
		cfg.RedirectURI = fallbackRedirectURI
	}
}
