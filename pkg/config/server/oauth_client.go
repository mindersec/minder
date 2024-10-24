// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
