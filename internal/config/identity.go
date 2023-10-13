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

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// IdentityConfig is the configuration for the identity provider
type IdentityConfig struct {
	Cli    CliIdentityConfig    `mapstructure:"cli"`
	Server ServerIdentityConfig `mapstructure:"server"`
}

// CliIdentityConfig is the configuration for the identity provider in the mediator cli
type CliIdentityConfig struct {
	// IssuerUrl is the base URL where the identity server is running
	IssuerUrl string `mapstructure:"issuer_url" default:"http://localhost:8081"`
	// Realm is the Keycloak realm that the client belongs to
	Realm string `mapstructure:"realm" default:"stacklok"`
	// ClientId is the client ID that identifies the mediator CLI
	ClientId string `mapstructure:"client_id" default:"mediator-cli"`
}

// ServerIdentityConfig is the configuration for the identity provider in mediator server
type ServerIdentityConfig struct {
	// IssuerUrl is the base URL where the identity server is running
	IssuerUrl string `mapstructure:"issuer_url" default:"http://localhost:8081"`
	// Realm is the Keycloak realm that the client belongs to
	Realm string `mapstructure:"realm" default:"stacklok"`
	// ClientId is the client ID that identifies the mediator server
	ClientId string `mapstructure:"client_id" default:"mediator-server"`
	// ClientSecret is the client secret for the mediator server
	ClientSecret string `mapstructure:"client_secret" default:"secret"`
	// ClientSecretFile is the location of a file containing the client secret for the mediator server (optional)
	ClientSecretFile string `mapstructure:"client_secret_file"`
}

// GetClientSecret returns the mediator-server client secret
func (sic *ServerIdentityConfig) GetClientSecret() (string, error) {
	if sic.ClientSecretFile != "" {
		data, err := os.ReadFile(filepath.Clean(sic.ClientSecretFile))
		if err != nil {
			return "", fmt.Errorf("failed to read mediator secret from file: %w", err)
		}
		return string(data), nil
	}
	return sic.ClientSecret, nil
}
