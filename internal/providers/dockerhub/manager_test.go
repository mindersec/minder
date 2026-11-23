// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

func TestNewDockerHubProviderClassManagerWebhookSecrets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	secretFile := filepath.Join(dir, "secret")
	require.NoError(t, os.WriteFile(secretFile, []byte("from-file"), 0o600))
	previousFile := filepath.Join(dir, "previous")
	require.NoError(t, os.WriteFile(previousFile, []byte("old-1\nold-2\n"), 0o600))

	tests := []struct {
		name         string
		cfg          *serverconfig.DockerHubConfig
		wantCurrent  string
		wantPrevious []string
		wantErr      bool
	}{
		{
			name: "no config",
		},
		{
			name: "inline secret",
			cfg: &serverconfig.DockerHubConfig{
				WebhookSecrets: serverconfig.WebhookSecrets{WebhookSecret: "inline"},
			},
			wantCurrent: "inline",
		},
		{
			name: "secret file and previous secrets",
			cfg: &serverconfig.DockerHubConfig{
				WebhookSecrets: serverconfig.WebhookSecrets{
					WebhookSecretFile:         secretFile,
					PreviousWebhookSecretFile: previousFile,
				},
			},
			wantCurrent:  "from-file",
			wantPrevious: []string{"old-1", "old-2"},
		},
		{
			name: "unreadable previous secrets file is not fatal",
			cfg: &serverconfig.DockerHubConfig{
				WebhookSecrets: serverconfig.WebhookSecrets{
					WebhookSecret:             "inline",
					PreviousWebhookSecretFile: filepath.Join(dir, "missing"),
				},
			},
			wantCurrent: "inline",
		},
		{
			name: "unreadable secret file",
			cfg: &serverconfig.DockerHubConfig{
				WebhookSecrets: serverconfig.WebhookSecrets{WebhookSecretFile: filepath.Join(dir, "missing")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m, err := NewDockerHubProviderClassManager(context.Background(), nil, nil, tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCurrent, m.currentWebhookSecret)
			assert.Equal(t, tt.wantPrevious, m.previousWebhookSecrets)
		})
	}
}

func TestWebhookToken(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	otherProviderID := uuid.New()

	m, err := NewDockerHubProviderClassManager(context.Background(), nil, nil, &serverconfig.DockerHubConfig{
		WebhookSecrets: serverconfig.WebhookSecrets{WebhookSecret: "current"},
	})
	require.NoError(t, err)

	token, err := m.webhookToken(providerID)
	require.NoError(t, err)
	assert.Len(t, token, 128, "expected a hex-encoded SHA-512 digest")

	again, err := m.webhookToken(providerID)
	require.NoError(t, err)
	assert.Equal(t, token, again, "the token for a provider should be stable")

	assert.True(t, m.validWebhookToken(providerID, token))
	assert.False(t, m.validWebhookToken(otherProviderID, token), "a token is only valid for its own provider")
	assert.False(t, m.validWebhookToken(providerID, ""))
	assert.False(t, m.validWebhookToken(providerID, token[:len(token)-1]))
}

func TestWebhookTokenRotation(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()

	oldManager, err := NewDockerHubProviderClassManager(context.Background(), nil, nil, &serverconfig.DockerHubConfig{
		WebhookSecrets: serverconfig.WebhookSecrets{WebhookSecret: "old"},
	})
	require.NoError(t, err)
	oldToken, err := oldManager.webhookToken(providerID)
	require.NoError(t, err)

	m := &providerClassManager{
		currentWebhookSecret:   "new",
		previousWebhookSecrets: []string{"older", "old"},
	}

	newToken, err := m.webhookToken(providerID)
	require.NoError(t, err)
	assert.NotEqual(t, oldToken, newToken, "rotating the secret should change the token")

	assert.True(t, m.validWebhookToken(providerID, newToken), "token under the current secret")
	assert.True(t, m.validWebhookToken(providerID, oldToken), "token under a previous secret")

	retired := &providerClassManager{currentWebhookSecret: "new"}
	assert.False(t, retired.validWebhookToken(providerID, oldToken),
		"a token is rejected once its secret is no longer listed as previous")
}

func TestWebhookTokenWithoutSecret(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()

	m, err := NewDockerHubProviderClassManager(context.Background(), nil, nil, nil)
	require.NoError(t, err)

	_, err = m.webhookToken(providerID)
	require.Error(t, err, "no token can be issued without a webhook secret")

	configured, err := NewDockerHubProviderClassManager(context.Background(), nil, nil, &serverconfig.DockerHubConfig{
		WebhookSecrets: serverconfig.WebhookSecrets{WebhookSecret: "current"},
	})
	require.NoError(t, err)
	token, err := configured.webhookToken(providerID)
	require.NoError(t, err)

	assert.False(t, m.validWebhookToken(providerID, token), "no token is valid without a webhook secret")
	assert.False(t, m.validWebhookToken(providerID, ""))
}
