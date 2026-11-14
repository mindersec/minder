// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_github "github.com/mindersec/minder/internal/providers/github/mock"
)

// TestGetAuthenticatorUsesProviderGate checks that the verifier takes its GHCR
// authenticator from the provider's class-gated GetContainerAuthenticator, not
// from the raw credential, so an App-backed provider (whose installation token
// GHCR rejects) fetches signatures anonymously.
func TestGetAuthenticatorUsesProviderGate(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		auth authn.Authenticator
	}{
		{name: "OAuth/PAT provider", auth: &authn.Basic{Username: "owner", Password: "token"}},
		{name: "App provider", auth: authn.Anonymous},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gh := mock_github.NewMockGitHub(ctrl)
			gh.EXPECT().GetContainerAuthenticator("owner").Return(tt.auth)
			// The raw credential must not be used directly.
			gh.EXPECT().GetCredential().Times(0)

			got := newContainerAuth(WithGitHubClient(gh)).getAuthenticator("owner")
			assert.Equal(t, tt.auth, got)
		})
	}
}

func TestGetAuthenticatorWithoutGitHubClient(t *testing.T) {
	t.Parallel()

	basic := &authn.Basic{Username: "u", Password: "p"}
	assert.Equal(t, basic, newContainerAuth(WithAuthenticator(basic)).getAuthenticator("owner"))
	assert.Equal(t, authn.Anonymous, newContainerAuth().getAuthenticator("owner"))
}
