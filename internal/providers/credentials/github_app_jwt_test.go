//
// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
