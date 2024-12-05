// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"bytes"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

func TestReadClientConfig(t *testing.T) {
	t.Parallel()

	clientCfgString := `---
grpc_server:
  host: "127.0.0.1"
  port: 8090

identity:
  cli:
    issuer_url: http://localhost:8081
    client_id: minder-cli
`
	cfgbuf := bytes.NewBufferString(clientCfgString)

	v := viper.New()

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "127.0.0.1", cfg.GRPCClientConfig.Host)
	require.Equal(t, 8090, cfg.GRPCClientConfig.Port)
	require.Equal(t, false, cfg.GRPCClientConfig.Insecure)
	require.Equal(t, "http://localhost:8081", cfg.Identity.CLI.IssuerUrl)
	require.Equal(t, "minder-cli", cfg.Identity.CLI.ClientId)
}

func TestReadClientConfigWithDefaults(t *testing.T) {
	t.Parallel()

	v := viper.New()

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	require.NoError(t, clientconfig.RegisterMinderClientFlags(v, flags), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "api.stacklok.com", cfg.GRPCClientConfig.Host)
	require.Equal(t, 443, cfg.GRPCClientConfig.Port)
	require.Equal(t, false, cfg.GRPCClientConfig.Insecure)
	require.Equal(t, "https://auth.stacklok.com", cfg.Identity.CLI.IssuerUrl)
	require.Equal(t, "minder-cli", cfg.Identity.CLI.ClientId)
}

func TestReadClientConfigWithConfigFileOverride(t *testing.T) {
	t.Parallel()

	clientCfgString := `---
grpc_server:
  host: "192.168.1.7"
identity:
  cli:
    issuer_url: http://localhost:1234
`
	cfgbuf := bytes.NewBufferString(clientCfgString)

	v := viper.New()

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	require.NoError(t, clientconfig.RegisterMinderClientFlags(v, flags), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "192.168.1.7", cfg.GRPCClientConfig.Host)
	require.Equal(t, 443, cfg.GRPCClientConfig.Port)
	require.Equal(t, false, cfg.GRPCClientConfig.Insecure)
	require.Equal(t, "http://localhost:1234", cfg.Identity.CLI.IssuerUrl)
	require.Equal(t, "minder-cli", cfg.Identity.CLI.ClientId)
}

func TestReadClientConfigWithCmdLineArgs(t *testing.T) {
	t.Parallel()

	v := viper.New()

	flags := pflag.NewFlagSet("test", pflag.PanicOnError)
	require.NoError(t, clientconfig.RegisterMinderClientFlags(v, flags), "Unexpected error")

	require.NoError(t, flags.Parse([]string{"--grpc-host=192.168.1.7", "--grpc-port=1234", "--identity-url=http://localhost:1654"}))
	t.Logf("Viper Configuration: %+v", v.AllSettings())

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](v)
	require.NoError(t, err, "Unexpected error")
	t.Logf("Read Configuration: %+v", cfg)

	require.Equal(t, "192.168.1.7", cfg.GRPCClientConfig.Host)
	require.Equal(t, 1234, cfg.GRPCClientConfig.Port)
	require.Equal(t, false, cfg.GRPCClientConfig.Insecure)
	require.Equal(t, "http://localhost:1654", cfg.Identity.CLI.IssuerUrl)
	require.Equal(t, "minder-cli", cfg.Identity.CLI.ClientId)
}

func TestReadClientConfigWithCmdLineArgsAndInputConfig(t *testing.T) {
	t.Parallel()

	clientCfgString := `---
grpc_server:
  host: "196.167.2.5"
identity:
  cli:
    issuer_url: http://localhost:4567
`
	cfgbuf := bytes.NewBufferString(clientCfgString)

	v := viper.New()

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	require.NoError(t, clientconfig.RegisterMinderClientFlags(v, flags), "Unexpected error")

	require.NoError(t, flags.Parse([]string{"--grpc-host=192.168.1.7", "--grpc-port=1234", "--identity-url=http://localhost:1654"}))

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "192.168.1.7", cfg.GRPCClientConfig.Host)
	require.Equal(t, 1234, cfg.GRPCClientConfig.Port)
	require.Equal(t, false, cfg.GRPCClientConfig.Insecure)
	require.Equal(t, "http://localhost:1654", cfg.Identity.CLI.IssuerUrl)
	require.Equal(t, "minder-cli", cfg.Identity.CLI.ClientId)
}
