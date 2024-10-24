// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package keystores_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/crypto/keystores"
	"github.com/mindersec/minder/pkg/config/server"
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

	keystore := keystores.NewKeyStoreFromMap(
		map[string][]byte{
			keyID: key,
		},
		keyID,
	)

	result, err := keystore.GetKey("")
	require.NoError(t, err)
	require.Equal(t, key, result)

	result, err = keystore.GetKey(keyID)
	require.NoError(t, err)
	require.Equal(t, key, result)
}

func TestLocalFileKeyStore_GetKeyEmptyStringNoFallback(t *testing.T) {
	t.Parallel()

	keystore := keystores.NewKeyStoreFromMap(
		map[string][]byte{
			keyID: key,
		},
		"",
	)

	_, err := keystore.GetKey("")
	require.ErrorContains(t, err, "empty key ID with no config defined")
}

const (
	keyID = "my_key"
)

var (
	key = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
)
