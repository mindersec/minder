// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package client contains the configuration for the minder cli
package client

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/config"
	"github.com/mindersec/minder/internal/constants"
)

// Config is the configuration for the minder cli
type Config struct {
	GRPCClientConfig config.GRPCClientConfig `mapstructure:"grpc_server" yaml:"grpc_server" json:"grpc_server"`
	Identity         IdentityConfigWrapper   `mapstructure:"identity" yaml:"identity" json:"identity"`
	// Project is the current project
	Project string `mapstructure:"project" yaml:"project" json:"project"`
}

// RegisterMinderClientFlags registers the flags for the minder cli
func RegisterMinderClientFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	if err := config.RegisterGRPCClientConfigFlags(v, flags); err != nil {
		return err
	}

	return registerClientIdentityConfigFlags(v, flags)
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
