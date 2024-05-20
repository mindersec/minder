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

package crypto

import (
	"testing"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto/algorithms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

//Test both the algorithm and the engine in one test suite
// TODO: if we add additional algorithms in future, we should split up testing

func TestNewFromCryptoConfig(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Config: map[string]any{
					"key_dir": "./testdata",
				},
			},
			Default: server.DefaultCrypto{
				KeyID:     "test_encryption_key",
				Algorithm: string(algorithms.Aes256Cfb),
			},
		},
	}
	_, err := NewEngineFromConfig(config)
	require.NoError(t, err)
}

func TestNewKeyLoadFail(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Auth: server.AuthConfig{
			TokenKey: "./testdata/non-existent-file",
		},
	}
	_, err := NewEngineFromConfig(config)
	require.ErrorContains(t, err, "failed to read token key file")
}

func TestNewKeyRejectsEmptyConfig(t *testing.T) {
	t.Parallel()

	config := &server.Config{}
	_, err := NewEngineFromConfig(config)
	require.ErrorContains(t, err, "no encryption keys configured")
}

func TestNewKeyRejectsBadAlgo(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Config: map[string]any{
					"key_dir": "./testdata",
				},
			},
			Default: server.DefaultCrypto{
				KeyID:     "test_encryption_key",
				Algorithm: "I'm a little teapot",
			},
		},
	}
	_, err := NewEngineFromConfig(config)
	require.ErrorIs(t, err, algorithms.ErrUnknownAlgorithm)
}

func TestNewKeyRejectsBadFallbackAlgo(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Config: map[string]any{
					"key_dir": "./testdata",
				},
			},
			Default: server.DefaultCrypto{
				KeyID:     "test_encryption_key",
				Algorithm: string(algorithms.Aes256Cfb),
			},
			Fallback: server.FallbackCrypto{
				Algorithms: []string{"what even is this?"},
			},
		},
	}
	_, err := NewEngineFromConfig(config)
	require.ErrorIs(t, err, algorithms.ErrUnknownAlgorithm)
}

func TestEncryptDecryptBytes(t *testing.T) {
	t.Parallel()

	const sampleData = "I'm a little teapot"
	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	encrypted, err := engine.EncryptString(sampleData)
	assert.Nil(t, err)
	decrypted, err := engine.DecryptString(encrypted)
	assert.Nil(t, err)
	assert.Equal(t, sampleData, decrypted)
}

func TestEncryptTooLarge(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	large := make([]byte, 34000000) // More than 32 MB
	_, err = engine.EncryptString(string(large))
	assert.ErrorIs(t, err, algorithms.ErrExceedsMaxSize)
}

func TestEncryptDecryptOAuthToken(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	oauthToken := oauth2.Token{AccessToken: "AUTH"}
	encryptedToken, err := engine.EncryptOAuthToken(&oauthToken)
	require.NoError(t, err)

	decrypted, err := engine.DecryptOAuthToken(encryptedToken)
	require.NoError(t, err)
	require.Equal(t, oauthToken, decrypted)
}

func TestDecryptEmpty(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	encryptedToken := EncryptedData{
		EncodedData: "",
	}

	_, err = engine.DecryptString(encryptedToken)
	require.ErrorContains(t, err, "cannot decrypt empty data")
}

func TestDecryptBadAlgorithm(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	encryptedToken := EncryptedData{
		Algorithm:   "I'm a little teapot",
		EncodedData: "abc",
		KeyVersion:  "",
	}
	require.NoError(t, err)

	_, err = engine.DecryptString(encryptedToken)
	require.ErrorIs(t, err, algorithms.ErrUnknownAlgorithm)
}

func TestDecryptBadEncoding(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	encryptedToken := EncryptedData{
		Algorithm: algorithms.Aes256Cfb,
		// Unicode snowman is _not_ a valid base64 character
		EncodedData: "☃☃☃☃☃☃☃☃☃☃☃☃☃☃☃",
		KeyVersion:  "",
	}
	require.NoError(t, err)

	_, err = engine.DecryptString(encryptedToken)
	require.ErrorContains(t, err, "error decoding secret")
}

func TestDecryptFailedDecryption(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	encryptedToken := EncryptedData{
		Algorithm: algorithms.Aes256Cfb,
		// too small of a value - will trigger the ciphertext length check
		EncodedData: "abcdef0123456789",
		KeyVersion:  "",
	}
	require.NoError(t, err)

	_, err = engine.DecryptString(encryptedToken)
	require.ErrorIs(t, err, ErrDecrypt)
}

var config = &server.Config{
	Auth: server.AuthConfig{
		TokenKey: "./testdata/test_encryption_key",
	},
}
