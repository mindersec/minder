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

// Package config contains a centralized structure for all configuration
// options.
package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/pkg/controlplane"
)

// Config is the top-level configuration structure.
type Config struct {
	HTTPServer controlplane.HTTPServerConfig `mapstructure:"http_server"`
	GRPCServer controlplane.GRPCServerConfig `mapstructure:"grpc_server"`
}

// ReadConfigFromViper reads the configuration from the given Viper instance.
// This will return the already-parsed and validated configuration, or an error.
func ReadConfigFromViper(v *viper.Viper) (*Config, error) {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// RegisterFlags registers all flags for the given Viper instance.
func RegisterFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	if err := controlplane.RegisterHTTPServerFlags(v, flags); err != nil {
		return err
	}

	return controlplane.RegisterGRPCServerFlags(v, flags)
}
