// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mindersec/minder/internal/crypto/algorithms"
	"github.com/mindersec/minder/pkg/config/server"
)

var config = &server.Config{
	Auth: server.AuthConfig{
		TokenKey: "./testdata/test_encryption_key",
	},
}

// Test both the algorithm and the engine in one test suite
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
				KeyID: "test_encryption_key",
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

	// instantiate engine with test_encryption_key
	// as default key
	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)

	// encrypt data with old config
	const sampleData = "Hello world!"
	encrypted, err := engine.EncryptString(sampleData)
	require.NoError(t, err)

	// Create new config where we introduce a new key, and the old key as a
	// fallback.
	newConfig := &server.Config{
		Crypto: server.CryptoConfig{
			KeyStore: server.KeyStoreConfig{
				Type: "local",
				Local: server.LocalKeyStoreConfig{
					KeyDir: "./testdata",
				},
			},
			Default: server.DefaultCrypto{
				KeyID: "test_encryption_key2",
			},
			Fallback: server.FallbackCrypto{
				KeyID: "test_encryption_key",
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
		Algorithm:   algorithms.Aes256Gcm,
		EncodedData: "abc",
		KeyVersion:  "test_encryption_key",
	}
	require.NoError(t, err)

	_, err = engine.DecryptString(encryptedToken)
	require.ErrorContains(t, err, "error decoding secret")
}

func TestDecryptNoVersionNoFallback(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	encryptedToken := EncryptedData{
		Algorithm: algorithms.Aes256Gcm,
		// Unicode snowman is _not_ a valid base64 character
		EncodedData: "☃☃☃☃☃☃☃☃☃☃☃☃☃☃☃",
		KeyVersion:  "",
	}
	require.NoError(t, err)

	_, err = engine.DecryptString(encryptedToken)
	require.ErrorContains(t, err, "empty key ID with no config defined")
}

func TestDecryptFailedDecryption(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)
	encryptedToken := EncryptedData{
		Algorithm: algorithms.Aes256Gcm,
		// too small of a value - will trigger the ciphertext length check
		EncodedData: "abcdef0123456789",
		KeyVersion:  "test_encryption_key",
	}
	require.NoError(t, err)

	_, err = engine.DecryptString(encryptedToken)
	require.ErrorIs(t, err, ErrDecrypt)
}

func TestEngineGeneratesUniqueSalts(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)

	const sampleData = "Super Secret Password"

	encrypted1, err := engine.EncryptString(sampleData)
	require.NoError(t, err)

	encrypted2, err := engine.EncryptString(sampleData)
	require.NoError(t, err)

	assert.NotEqual(t, encrypted1.EncodedData, encrypted2.EncodedData)
	assert.NotEqual(t, encrypted1.Salt, encrypted2.Salt)

	decrypted1, err := engine.DecryptString(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, sampleData, decrypted1)
}

func TestEngineRoutesLegacyCFBDecryption(t *testing.T) {
	t.Parallel()

	engine, err := NewEngineFromConfig(config)
	require.NoError(t, err)

	const sampleData = "Legacy CFB Secret"
	cfbAlgo := &algorithms.AES256CFBAlgorithm{}

	keyBytes, err := os.ReadFile(config.Auth.TokenKey)
	require.NoError(t, err)

	testSalt := []byte("1234567890123456")
	rawCiphertext, err := cfbAlgo.Encrypt([]byte(sampleData), keyBytes, testSalt)
	require.NoError(t, err)

	legacyPayload := EncryptedData{
		Algorithm:   algorithms.Aes256Cfb,
		EncodedData: base64.StdEncoding.EncodeToString(rawCiphertext),
		KeyVersion:  "test_encryption_key",
		Salt:        base64.StdEncoding.EncodeToString(testSalt),
	}

	decrypted, err := engine.DecryptString(legacyPayload)
	require.NoError(t, err)
	assert.Equal(t, sampleData, decrypted)
}

func TestGCM_NoDoubleStorage(t *testing.T) {
	t.Parallel()

	algo := &algorithms.AES256GCMAlgorithm{}
	validKey := []byte("YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE=")
	salt := []byte("1234567890123456")
	plaintext := []byte("hello")

	ciphertext, err := algo.Encrypt(plaintext, validKey, salt)
	require.NoError(t, err)

	// Verify nonce isn't prepended (Length = plaintext + GCM Tag)
	assert.Equal(t, len(plaintext)+16, len(ciphertext))
}

func TestGCM_SafelyHandlesKeystoreNewlines(t *testing.T) {
	t.Parallel()

	algo := &algorithms.AES256GCMAlgorithm{}
	dirtyKey := []byte("YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE=\n")
	salt := []byte("1234567890123456")
	plaintext := []byte("testing newlines")

	ciphertext, err := algo.Encrypt(plaintext, dirtyKey, salt)
	require.NoError(t, err)

	decrypted, err := algo.Decrypt(ciphertext, dirtyKey, salt)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCFB_LegacySaltFallback(t *testing.T) {
	t.Parallel()

	algo := &algorithms.AES256CFBAlgorithm{}
	key := []byte("my_raw_password_string\n")
	plaintext := []byte("legacy fallback test")

	var emptySalt []byte
	ciphertext, err := algo.Encrypt(plaintext, key, emptySalt)
	require.NoError(t, err)

	decrypted, err := algo.Decrypt(ciphertext, key, emptySalt)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCFB_DynamicSalt(t *testing.T) {
	t.Parallel()

	algo := &algorithms.AES256CFBAlgorithm{}

	key := []byte("a_very_secure_passphrase\n")
	salt := []byte("dynamic_salt_123")
	plaintext := []byte("testing dynamic salt with cfb")

	ciphertext, err := algo.Encrypt(plaintext, key, salt)
	require.NoError(t, err)

	decrypted, err := algo.Decrypt(ciphertext, key, salt)
	require.NoError(t, err)

	assert.Equal(t, string(plaintext), string(decrypted), "CFB should successfully encrypt/decrypt using a dynamic salt")

	wrongSalt := []byte("wrong_salt_45678")
	wrongDecrypted, _ := algo.Decrypt(ciphertext, key, wrongSalt)
	assert.NotEqual(t, string(plaintext), string(wrongDecrypted), "Decryption with the wrong salt should produce garbage data")
}
