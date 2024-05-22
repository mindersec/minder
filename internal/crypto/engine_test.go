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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto/algorithms"
)

//Test both the algorithm and the engine in one test suite
// TODO: if we add additional algorithms in future, we should split up testing

func TestNewFromCryptoConfig(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Local: server.LocalKeyStoreConfig{
					KeyDir: "./testdata",
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

func TestNewKeystoreFail(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Auth: server.AuthConfig{
			TokenKey: "./testdata/non-existent-file",
		},
	}
	_, err := NewEngineFromConfig(config)
	require.ErrorContains(t, err, "unable to create keystore")
}

func TestNewRejectsEmptyConfig(t *testing.T) {
	t.Parallel()

	config := &server.Config{}
	_, err := NewEngineFromConfig(config)
	require.ErrorContains(t, err, "no encryption keys configured")
}

func TestNewRejectsBadAlgo(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Local: server.LocalKeyStoreConfig{
					KeyDir: "./testdata",
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

func TestNewRejectsBadFallbackAlgo(t *testing.T) {
	t.Parallel()

	config := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Local: server.LocalKeyStoreConfig{
					KeyDir: "./testdata",
				},
			},
			Default: server.DefaultCrypto{
				KeyID:     "test_encryption_key",
				Algorithm: string(algorithms.Aes256Cfb),
			},
			Fallback: server.FallbackCrypto{
				Algorithm: "what even is this?",
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

func TestFallbackDecrypt(t *testing.T) {
	t.Parallel()

	// instantiate engine with CFB as default algorithm
	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)

	// encrypt data with old config
	const sampleData = "Hello world!"
	encrypted, err := engine.EncryptString(sampleData)
	require.NoError(t, err)

	// Create new config where we introduce a new default algorithm
	// and make CFB the fallback.
	newConfig := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Local: server.LocalKeyStoreConfig{
					KeyDir: "./testdata",
				},
			},
			Default: server.DefaultCrypto{
				KeyID:     "test_encryption_key",
				Algorithm: string(algorithms.Aes256Gcm),
			},
			Fallback: server.FallbackCrypto{
				Algorithm: string(algorithms.Aes256Cfb),
				KeyID:     "test_encryption_key2",
			},
		},
	}

	// instantiate new engine
	newEngine, err := NewEngineFromConfig(newConfig)
	require.NoError(t, err)

	// decrypt data from old engine with new engine
	// this validates that the fallback works as expected
	decrypted, err := newEngine.DecryptString(encrypted)
	assert.Nil(t, err)
	assert.Equal(t, sampleData, decrypted)

	// encrypt the data with the new engine - assert it is not the same as the old result
	newEncrypted, err := newEngine.EncryptString(sampleData)
	require.NoError(t, err)
	require.NotEqualf(t, newEncrypted, encrypted, "two encrypted values expected to be different but are not")
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
