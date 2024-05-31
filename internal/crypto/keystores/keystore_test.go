// Copyright 2024 Stacklok, Inc.
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

package keystores_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto/keystores"
)

func TestNewKeyStoreFromConfig(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name          string
		Config        server.CryptoConfig
		ExpectedError string
	}{
		{
			Name: "NewKeyStoreFromConfig rejects empty keystore config",
			Config: server.CryptoConfig{
				KeyStore: server.KeyStoreConfig{
					Type: keystores.LocalKeyStore,
				},
			},
			ExpectedError: "key directory not defined in keystore config",
		},
		{
			Name: "NewKeyStoreFromConfig rejects invalid keystore type",
			Config: server.CryptoConfig{
				KeyStore: server.KeyStoreConfig{
					Type: "derp",
				},
			},
			ExpectedError: "unexpected keystore type",
		},
		{
			Name: "NewKeyStoreFromConfig returns error when key cannot be read",
			Config: server.CryptoConfig{
				KeyStore: server.KeyStoreConfig{
					Type: keystores.LocalKeyStore,
					Local: server.LocalKeyStoreConfig{
						KeyDir: "../testdata",
					},
				},
				Default: server.DefaultCrypto{
					KeyID: "not-a-valid-file",
				},
			},
			ExpectedError: "unable to read key",
		},
		{
			Name: "NewKeyStoreFromConfig successfully creates keystore",
			Config: server.CryptoConfig{
				KeyStore: server.KeyStoreConfig{
					Type: keystores.LocalKeyStore,
					Local: server.LocalKeyStoreConfig{
						KeyDir: "../testdata",
					},
				},
				Default: server.DefaultCrypto{
					KeyID: "test_encryption_key",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			_, err := keystores.NewKeyStoreFromConfig(scenario.Config)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

// These tests are trivial, avoiding use of scenario structure
func TestLocalFileKeyStore_GetKey(t *testing.T) {
	t.Parallel()

	keyID := "my_key"
	key := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	keystore := keystores.NewKeyStoreFromMap(
		map[string][]byte{
			keyID: key,
		},
		"",
	)

	result, err := keystore.GetKey(keyID)
	require.NoError(t, err)
	require.Equal(t, key, result)

	_, err = keystore.GetKey("foobar")
	require.ErrorIs(t, err, keystores.ErrUnknownKeyID)
}

func TestLocalFileKeyStore_GetKeyEmptyString(t *testing.T) {
	t.Parallel()

	keyID := "my_key"
	key := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	keystore := keystores.NewKeyStoreFromMap(
		map[string][]byte{
			keyID: key,
		},
		keyID,
	)

	result, err := keystore.GetKey("")
	require.NoError(t, err)
	require.Equal(t, key, result)

	_, err = keystore.GetKey("foobar")
	require.ErrorIs(t, err, keystores.ErrUnknownKeyID)
}

func TestLocalFileKeyStore_GetKeyEmptyStringNoFallback(t *testing.T) {
	t.Parallel()

	keyID := "my_key"
	key := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	keystore := keystores.NewKeyStoreFromMap(
		map[string][]byte{
			keyID: key,
		},
		"",
	)

	_, err := keystore.GetKey("")
	require.ErrorContains(t, err, "empty key ID with no config defined")
}
