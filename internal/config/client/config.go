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

// Package client contains the configuration for the minder cli
package client

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	"github.com/stacklok/minder/internal/constants"
)

// Config is the configuration for the minder cli
type Config struct {
	GRPCClientConfig GRPCClientConfig      `mapstructure:"grpc_server"`
	Identity         IdentityConfigWrapper `mapstructure:"identity"`
}

// GRPCClientConfig is the configuration for the minder cli to connect to gRPC server
type GRPCClientConfig struct {
	// Host is the host to connect to
	Host string `mapstructure:"host" default:"api.stacklok.com"`

	// Port is the port to connect to
	Port int `mapstructure:"port" default:"443"`

	// Insecure is whether to allow establishing insecure connections
	Insecure bool `mapstructure:"insecure" default:"false"`
}

// RegisterMinderClientFlags registers the flags for the minder cli
func RegisterMinderClientFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	if err := registerGRPCClientConfigFlags(v, flags); err != nil {
		return err
	}

	return registerClientIdentityConfigFlags(v, flags)
}

// registerGRPCClientConfigFlags registers the flags for the gRPC client
func registerGRPCClientConfigFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := config.BindConfigFlag(v, flags, "grpc_server.host", "grpc-host", constants.MinderGRPCHost,
		"Server host", flags.String)
	if err != nil {
		return err
	}

	err = config.BindConfigFlag(v, flags, "grpc_server.port", "grpc-port", 443,
		"Server port", flags.Int)
	if err != nil {
		return err
	}

	return config.BindConfigFlag(v, flags, "grpc_server.insecure", "grpc-insecure", false,
		"Allow establishing insecure connections", flags.Bool)
}

// registerClientIdentityConfigFlags registers the flags for the client identity
func registerClientIdentityConfigFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := config.BindConfigFlag(v, flags, "identity.cli.issuer_url", "identity-url", constants.IdentitySeverURL,
		"Identity server issuer URL", flags.String)
	if err != nil {
		return err
	}

	return config.BindConfigFlag(v, flags, "identity.cli.client_id", "identity-client", "minder-cli",
		"Identity server client ID", flags.String)
}
