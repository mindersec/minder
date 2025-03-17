// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package client contains the configuration for the minder cli
package client

import (
	"crypto/tls"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mindersec/minder/internal/constants"
	"github.com/mindersec/minder/pkg/config"
)

// Config is the configuration for the minder cli
type Config struct {
	GRPCClientConfig GRPCClientConfig      `mapstructure:"grpc_server" yaml:"grpc_server" json:"grpc_server"`
	Identity         IdentityConfigWrapper `mapstructure:"identity" yaml:"identity" json:"identity"`
	// Project is the current project
	Project string `mapstructure:"project" yaml:"project" json:"project"`
}

// RegisterMinderClientFlags registers the flags for the minder cli
func RegisterMinderClientFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	if err := RegisterGRPCClientConfigFlags(v, flags); err != nil {
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

// GRPCClientConfig is the configuration for a service to connect to minder gRPC server
type GRPCClientConfig struct {
	// Host is the host to connect to
	Host string `mapstructure:"host" yaml:"host" json:"host" default:"api.stacklok.com"`

	// Port is the port to connect to
	Port int `mapstructure:"port" yaml:"port" json:"port" default:"443"`

	// Insecure is whether to allow establishing insecure connections
	Insecure bool `mapstructure:"insecure" yaml:"insecure" json:"insecure" default:"false"`
}

// RegisterGRPCClientConfigFlags registers the flags for the gRPC client
func RegisterGRPCClientConfigFlags(v *viper.Viper, flags *pflag.FlagSet) error {
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

// GetGRPCAddress returns the formatted GRPC address of the server.
func (c GRPCClientConfig) GetGRPCAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// TransportCredentialsOption returns a gRPC dial option appropriate to the
// configuration (either TLS with correct hostname, or without verification).
func (c GRPCClientConfig) TransportCredentialsOption() grpc.DialOption {
	insecureDefault := c.Host == "localhost" || c.Host == "127.0.0.1" || c.Host == "::1"
	allowInsecure := c.Insecure || insecureDefault

	if allowInsecure {
		return grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
		ServerName: c.Host,
	}))
}
