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

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
)

// IdentityConfigWrapper is the configuration for the identity provider
type IdentityConfigWrapper struct {
	Server IdentityConfig `mapstructure:"server"`
}

// IdentityConfig is the configuration for the identity provider in minder server
type IdentityConfig struct {
	// IssuerUrl is the base URL where the identity server is running
	IssuerUrl string `mapstructure:"issuer_url" default:"http://localhost:8081"`
	// ClientId is the client ID that identifies the minder server
	ClientId string `mapstructure:"client_id" default:"minder-server"`
	// ClientSecret is the client secret for the minder server
	ClientSecret string `mapstructure:"client_secret" default:"secret"`
	// ClientSecretFile is the location of a file containing the client secret for the minder server (optional)
	ClientSecretFile string `mapstructure:"client_secret_file"`
}

// GetClientSecret returns the minder-server client secret
func (sic *IdentityConfig) GetClientSecret() (string, error) {
	if sic.ClientSecretFile != "" {
		data, err := os.ReadFile(filepath.Clean(sic.ClientSecretFile))
		if err != nil {
			return "", fmt.Errorf("failed to read minder secret from file: %w", err)
		}
		return string(data), nil
	}
	return sic.ClientSecret, nil
}

// RegisterIdentityFlags registers the flags for the identity server
func RegisterIdentityFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	return config.BindConfigFlag(v, flags, "identity.server.issuer_url", "issuer-url", "",
		"The base URL where the identity server is running", flags.String)
}

func (ic *IdentityConfig) Issuer() url.URL {
	u, err := url.Parse(ic.IssuerUrl)
	if err != nil {
		panic("invalid issuer URL")
	}
	return *u
}