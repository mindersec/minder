//
// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	crand "crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
)

func TestCreateGitHubAppJWT(t *testing.T) {
	t.Parallel()
	privateKey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err, "failed to generate private key")

	expectedAppId := int64(123456)

	token, err := CreateGitHubAppJWT(expectedAppId, privateKey)
	require.NoError(t, err, "failed to generate JWT")

	parsed, err := jwt.ParseString(token)
	if err != nil {
		return
	}
	require.Equal(t, expectedAppId, parsed.Issuer())
}
