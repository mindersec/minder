// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuth2TokenCredential(t *testing.T) {
	t.Parallel()

	token := "oauth2-access-token-xyz"
	cred := NewOAuth2TokenCredential(token)
	require.NotNil(t, cred)

	ts := cred.GetAsOAuth2TokenSource()
	require.NotNil(t, ts)

	tok, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, token, tok.AccessToken)
}
