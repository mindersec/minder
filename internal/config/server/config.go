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

// Package server contains a centralized structure for all configuration
// options.
package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
)

// Config is the top-level configuration structure.
type Config struct {
	HTTPServer      HTTPServerConfig      `mapstructure:"http_server"`
	GRPCServer      GRPCServerConfig      `mapstructure:"grpc_server"`
	MetricServer    MetricServerConfig    `mapstructure:"metric_server"`
	LoggingConfig   LoggingConfig         `mapstructure:"logging"`
	Tracing         TracingConfig         `mapstructure:"tracing"`
	Metrics         MetricsConfig         `mapstructure:"metrics"`
	Flags           FlagsConfig           `mapstructure:"flags"`
	Database        config.DatabaseConfig `mapstructure:"database"`
	Identity        IdentityConfigWrapper `mapstructure:"identity"`
	Auth            AuthConfig            `mapstructure:"auth"`
	WebhookConfig   WebhookConfig         `mapstructure:"webhook-config"`
	Events          EventConfig           `mapstructure:"events"`
	Authz           AuthzConfig           `mapstructure:"authz"`
	Provider        ProviderConfig        `mapstructure:"provider"`
	Marketplace     MarketplaceConfig     `mapstructure:"marketplace"`
	DefaultProfiles DefaultProfilesConfig `mapstructure:"default_profiles"`
	Crypto          CryptoConfig          `mapstructure:"crypto"`
	Email           EmailConfig           `mapstructure:"email"`
}

// DefaultConfigForTest returns a configuration with all the struct defaults set,
// but no other changes.
func DefaultConfigForTest() *Config {
	v := viper.New()
	SetViperDefaults(v)
	c, err := config.ReadConfigFromViper[Config](v)
	if err != nil {
		panic(fmt.Sprintf("Failed to read default config: %v", err))
	}
	return c
}

// SetViperDefaults sets the default values for the configuration to be picked
// up by viper
func SetViperDefaults(v *viper.Viper) {
	v.SetEnvPrefix("minder")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	config.SetViperStructDefaults(v, "", Config{})
}

func fileOrArg(file, arg, desc string) (string, error) {
	if file != "" {
		data, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return "", fmt.Errorf("failed to read %s from file: %w", desc, err)
		}
		return string(data), nil
	}
	return arg, nil
}
