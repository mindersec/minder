// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/config"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

func readServerConfig(t *testing.T, cfgstr string) *serverconfig.Config {
	t.Helper()

	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(bytes.NewBufferString(cfgstr)))

	cfg, err := config.ReadConfigFromViper[serverconfig.Config](v)
	require.NoError(t, err)
	return cfg
}

func TestDockerHubConfigWebhookSecrets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	secretFile := filepath.Join(dir, "secret")
	require.NoError(t, os.WriteFile(secretFile, []byte("from-file"), 0o600))
	previousFile := filepath.Join(dir, "previous")
	require.NoError(t, os.WriteFile(previousFile, []byte("old-1\nold-2 old-3\n"), 0o600))

	cfg := readServerConfig(t, `---
provider:
  dockerhub:
    webhook_secret: "inline-secret"
    webhook_secret_file: "`+secretFile+`"
    previous_webhook_secret_file: "`+previousFile+`"
`)

	require.NotNil(t, cfg.Provider.DockerHub)
	require.Equal(t, "inline-secret", cfg.Provider.DockerHub.WebhookSecret)

	// As for GitLab, the secret file takes precedence over the inline secret.
	secret, err := cfg.Provider.DockerHub.GetWebhookSecret()
	require.NoError(t, err)
	require.Equal(t, "from-file", secret)

	previous, err := cfg.Provider.DockerHub.GetPreviousWebhookSecrets()
	require.NoError(t, err)
	require.Equal(t, []string{"old-1", "old-2", "old-3"}, previous)
}

func TestDockerHubConfigInlineWebhookSecret(t *testing.T) {
	t.Parallel()

	cfg := readServerConfig(t, `---
provider:
  dockerhub:
    webhook_secret: "inline-secret"
`)

	require.NotNil(t, cfg.Provider.DockerHub)
	secret, err := cfg.Provider.DockerHub.GetWebhookSecret()
	require.NoError(t, err)
	require.Equal(t, "inline-secret", secret)
}

func TestDockerHubConfigAbsent(t *testing.T) {
	t.Parallel()

	cfg := readServerConfig(t, `---
provider:
  gitlab:
    webhook_secret: "gitlab-secret"
`)

	require.Nil(t, cfg.Provider.DockerHub)
	require.NotNil(t, cfg.Provider.GitLab)
}
