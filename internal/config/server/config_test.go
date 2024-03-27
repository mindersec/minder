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

package server_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
)

func TestReadValidConfig(t *testing.T) {
	t.Parallel()

	cfgstr := `---
http_server:
  host:	"myhost"
  port:	8666
grpc_server:
  host:	"myhost"
  port:	8667
metric_server:
  host:	"myhost"
  port:	8668
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[serverconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "myhost", cfg.HTTPServer.Host)
	require.Equal(t, 8666, cfg.HTTPServer.Port)
	require.Equal(t, "myhost", cfg.GRPCServer.Host)
	require.Equal(t, 8667, cfg.GRPCServer.Port)
	require.Equal(t, "myhost", cfg.MetricServer.Host)
	require.Equal(t, 8668, cfg.MetricServer.Port)
}

func TestReadConfigWithDefaults(t *testing.T) {
	t.Parallel()

	cfgstr := `---
http_server:
grpc_server:
metric_server:
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	require.NoError(t, serverconfig.RegisterServerFlags(v, flags), "Unexpected error")

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[serverconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "", cfg.HTTPServer.Host)
	require.Equal(t, 8080, cfg.HTTPServer.Port)
	require.Equal(t, "", cfg.GRPCServer.Host)
	require.Equal(t, 8090, cfg.GRPCServer.Port)
	require.Equal(t, "", cfg.MetricServer.Host)
	require.Equal(t, 9090, cfg.MetricServer.Port)
}

func TestReadConfigWithCommandLineArgOverrides(t *testing.T) {
	t.Parallel()

	cfgstr := `---
http_server:
  host:	"myhost"
  port:	8666
grpc_server:
  host:	"myhost"
  port:	8667
metric_server:
  host:	"myhost"
  port:	8668
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	require.NoError(t, serverconfig.RegisterServerFlags(v, flags), "Unexpected error")

	require.NoError(t, flags.Parse([]string{"--http-host=foo", "--http-port=1234", "--grpc-host=bar", "--grpc-port=5678", "--metric-host=var", "--metric-port=6679"}))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[serverconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "foo", cfg.HTTPServer.Host)
	require.Equal(t, 1234, cfg.HTTPServer.Port)
	require.Equal(t, "bar", cfg.GRPCServer.Host)
	require.Equal(t, 5678, cfg.GRPCServer.Port)
	require.Equal(t, "var", cfg.MetricServer.Host)
	require.Equal(t, 6679, cfg.MetricServer.Port)
}

func TestMergeDBConfig(t *testing.T) {
	t.Parallel()

	cfgstr := `---
events:
  sql:
    connection:
      dbhost: "myhost"
      # dbport is not set
      dbuser: "myuser"
      # Don't set dbpass, etc.
`
	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	// Without SetViperDefaults calling v.SetDefault on each field, the default
	// values don't merge correctly.
	serverconfig.SetViperDefaults(v)

	require.NoError(t, serverconfig.RegisterServerFlags(v, flags), "Unexpected error")
	require.NoError(t, config.RegisterDatabaseFlags(v, flags), "Unexpected error")

	// Make sure that `database.dbhost` doesn't affect events
	require.NoError(t, flags.Parse([]string{"--db-host=production-host"}))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[serverconfig.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "myhost", cfg.Events.SQLPubSub.Connection.Host)
	require.Equal(t, 5432, cfg.Events.SQLPubSub.Connection.Port)
	require.Equal(t, "myuser", cfg.Events.SQLPubSub.Connection.User)
	require.Equal(t, "postgres", cfg.Events.SQLPubSub.Connection.Password)
	require.Equal(t, "minder", cfg.Events.SQLPubSub.Connection.Name)
	require.Equal(t, "disable", cfg.Events.SQLPubSub.Connection.SSLMode)
}

func TestReadDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := serverconfig.DefaultConfigForTest()
	require.Equal(t, "debug", cfg.LoggingConfig.Level)
	require.Equal(t, "minder", cfg.Database.Name)
	require.Equal(t, "./.ssh/token_key_passphrase", cfg.Auth.TokenKey)
}

func TestReadAuthConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		mutate   func(*testing.T, *serverconfig.AuthConfig)
		keyError string
	}{
		{
			name: "valid config",
		},
		{
			name: "missing keys",
			mutate: func(t *testing.T, cfg *serverconfig.AuthConfig) {
				t.Helper()
				require.NoError(t, os.Remove(cfg.TokenKey))
			},
			keyError: "no such file or directory",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmpdir := t.TempDir()
			cfg := serverconfig.AuthConfig{
				TokenKey: filepath.Join(tmpdir, "token_key"),
			}
			if err := os.WriteFile(cfg.TokenKey, []byte("test"), 0600); err != nil {
				t.Fatalf("Error generating access token key pair: %v", err)
			}

			if tc.mutate != nil {
				tc.mutate(t, &cfg)
			}

			if _, err := cfg.GetTokenKey(); !errMatches(err, tc.keyError) {
				t.Errorf("Expected error containing %q, but got %v", tc.keyError, err)
			}
		})
	}
}

func errMatches(got error, want string) bool {
	if want == "" {
		return got == nil
	}
	return strings.Contains(got.Error(), want)
}

const (
	viperPath  = "test.path"
	cmdLineArg = "test-arg"
	help       = "test help"
)

func TestBindConfigFlagStringWithArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := "test"

	err := config.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.String)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{"--" + cmdLineArg + "=foo"}))
	require.Equal(t, "foo", v.GetString(viperPath))
}

func TestBindConfigFlagStringWithDefaultArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := "test"

	err := config.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.String)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{}))
	require.Equal(t, defaultValue, v.GetString(viperPath))
}

func TestBindConfigFlagIntWithArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := 123

	err := config.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.Int)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{"--" + cmdLineArg + "=456"}))
	require.Equal(t, 456, v.GetInt(viperPath))
}

func TestBindConfigFlagIntWithDefaultArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := 123

	err := config.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.Int)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{}))
	require.Equal(t, 123, v.GetInt(viperPath))
}
