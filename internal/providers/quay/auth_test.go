// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/manager"
)

func TestNewOAuthConfig(t *testing.T) {
	t.Parallel()

	pcm := &providerClassManager{}

	// Quay only supports the user-input (token) flow, so the OAuth2
	// authorization code flow is unsupported, not silently misconfigured.
	cfg, err := pcm.NewOAuthConfig(Quay, true)
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()

	pcm := &providerClassManager{}

	t.Run("valid token string", func(t *testing.T) {
		t.Parallel()
		err := pcm.ValidateCredentials(context.Background(), "a-valid-token", &manager.CredentialVerifyParams{})
		assert.NoError(t, err)
	})

	t.Run("empty token string", func(t *testing.T) {
		t.Parallel()
		err := pcm.ValidateCredentials(context.Background(), "", &manager.CredentialVerifyParams{})
		assert.Error(t, err)
	})

	t.Run("oauth2 token credential is unsupported", func(t *testing.T) {
		t.Parallel()
		// Quay only supports the user-input flow, so ValidateCredentials
		// never receives an OAuth2-derived credential in practice; this
		// asserts it is rejected rather than silently accepted.
		err := pcm.ValidateCredentials(
			context.Background(), credentials.NewOAuth2TokenCredential("a-valid-token"), &manager.CredentialVerifyParams{},
		)
		assert.Error(t, err)
	})

	t.Run("unsupported credential type", func(t *testing.T) {
		t.Parallel()
		err := pcm.ValidateCredentials(context.Background(), 42, &manager.CredentialVerifyParams{})
		assert.Error(t, err)
	})
}
