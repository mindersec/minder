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
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-jwt/jwt/v4"

	"github.com/stacklok/minder/internal/config"
)

// GitHubAppConfig is the configuration for the GitHub App providers
type GitHubAppConfig struct {
	// AppName is the name of the GitHub App
	AppName string `mapstructure:"app_name"`
	// AppID is the ID of the GitHub App
	AppID int64 `mapstructure:"app_id" default:"0"`
	// UserID is the ID of the GitHub App user
	UserID int64 `mapstructure:"user_id" default:"0"`
	// PrivateKey is the path to the GitHub App's private key in PEM format
	PrivateKey string `mapstructure:"private_key"`
	// WebhookSecret is the GitHub App's webhook secret
	WebhookSecret string `mapstructure:"webhook_secret"`
	// WebhookSecretFile is the location of the file containing the GitHub App's webhook secret
	WebhookSecretFile string `mapstructure:"webhook_secret_file"`
	// FallbackToken is the fallback token to use when listing packages
	FallbackToken string `mapstructure:"fallback_token"`
	// FallbackTokenFile is the location of the file containing the fallback token to use when listing packages
	FallbackTokenFile string `mapstructure:"fallback_token_file"`
}

// GetPrivateKey returns the GitHub App's private key
func (ghcfg *GitHubAppConfig) GetPrivateKey() (*rsa.PrivateKey, error) {
	privateKeyBytes, err := config.ReadKey(ghcfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	return privateKey, err
}

// GetWebhookSecret returns the GitHub App's webhook secret
func (ghcfg *GitHubAppConfig) GetWebhookSecret() (string, error) {
	if ghcfg.WebhookSecretFile != "" {
		data, err := os.ReadFile(filepath.Clean(ghcfg.WebhookSecretFile))
		if err != nil {
			return "", fmt.Errorf("failed to read GitHub App webhook secret from file: %w", err)
		}
		return string(data), nil
	}
	return ghcfg.WebhookSecret, nil
}

// GetFallbackToken returns the GitHub App's fallback token
func (ghcfg *GitHubAppConfig) GetFallbackToken() (string, error) {
	if ghcfg.FallbackTokenFile != "" {
		data, err := os.ReadFile(filepath.Clean(ghcfg.FallbackTokenFile))
		if err != nil {
			return "", fmt.Errorf("failed to read GitHub App fallback token from file: %w", err)
		}
		return string(data), nil
	}
	return ghcfg.FallbackToken, nil
}
