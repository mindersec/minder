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

package config_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/util"
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

	cfg, err := config.ReadConfigFromViper(v)
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

	require.NoError(t, config.RegisterServerFlags(v, flags), "Unexpected error")

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper(v)
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

	require.NoError(t, config.RegisterServerFlags(v, flags), "Unexpected error")

	require.NoError(t, flags.Parse([]string{"--http-host=foo", "--http-port=1234", "--grpc-host=bar", "--grpc-port=5678", "--metric-host=var", "--metric-port=6679"}))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper(v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "foo", cfg.HTTPServer.Host)
	require.Equal(t, 1234, cfg.HTTPServer.Port)
	require.Equal(t, "bar", cfg.GRPCServer.Host)
	require.Equal(t, 5678, cfg.GRPCServer.Port)
	require.Equal(t, "var", cfg.MetricServer.Host)
	require.Equal(t, 6679, cfg.MetricServer.Port)
}

func TestReadDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfigForTest()
	require.Equal(t, "debug", cfg.LoggingConfig.Level)
	require.Equal(t, "mediator", cfg.Database.Name)
	require.Equal(t, "./.ssh/token_key_passphrase", cfg.Auth.TokenKey)
}

func TestReadAuthConfig(t *testing.T) {
	t.Parallel()
	tmpdir := t.TempDir()
	config := config.AuthConfig{
		AccessTokenPrivateKey:  filepath.Join(tmpdir, "access_token_private.pem"),
		AccessTokenPublicKey:   filepath.Join(tmpdir, "access_token_public.pem"),
	}
	testCases := []struct {
		name string
		mutate func()
		publicKeyError string
		privateKeyError string
	}{
		{
			name: "valid config",
		},
		{
			name: "missing keys",
			mutate: func() {
				os.Remove(config.AccessTokenPrivateKey)
				os.Remove(config.AccessTokenPublicKey)
			},
			publicKeyError: "no such file or directory",
			privateKeyError: "no such file or directory",
		},
		{
			name: "bad formats",
			mutate: func() {
				privateBytes, err := os.ReadFile(config.AccessTokenPrivateKey)
				if err != nil {
					t.Fatalf("Unable to open %q: %v", config.AccessTokenPrivateKey, err)
				}
				if err := os.WriteFile(config.AccessTokenPublicKey, privateBytes, 0600); err != nil {
					t.Fatalf("Unable to overwrite %q: %v", config.AccessTokenPublicKey, err)
				}
				if err := os.WriteFile(config.AccessTokenPrivateKey, []byte("bad format"), 0600); err != nil {
					t.Fatalf("Unable to overwrite %q: %v", config.AccessTokenPrivateKey, err)
				}
			},
			publicKeyError: "could not find PKCS1 or PKIX public key in",
			privateKeyError: "Key must be a PEM encoded PKCS1 or PKCS8 key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := util.RandomKeypairFile(2048, config.AccessTokenPrivateKey, config.AccessTokenPublicKey)
			if err != nil {
				t.Fatalf("Error generating access token key pair: %v", err)
			}

			if tc.mutate != nil {
				tc.mutate()
			}
			
			if _, err := config.GetAccessTokenPrivateKey(); !errMatches(err, tc.privateKeyError) {
				t.Errorf("Expected error containing %q, but got %v", tc.privateKeyError, err)
			}
			if _, err := config.GetAccessTokenPublicKey(); !errMatches(err, tc.publicKeyError) {
				t.Errorf("Expected error containing %q, but got %v", tc.publicKeyError, err)
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
