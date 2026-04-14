// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package crypto provides cryptographic functions
package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/oauth2"

	"github.com/mindersec/minder/internal/crypto/algorithms"
	"github.com/mindersec/minder/internal/crypto/keystores"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// Engine provides all functions to encrypt and decrypt data
type Engine interface {
	// EncryptOAuthToken takes an OAuth2 token, serializes to JSON and encrypts it.
	EncryptOAuthToken(token *oauth2.Token) (EncryptedData, error)
	// DecryptOAuthToken takes an OAuth2 token encrypted using EncryptOAuthToken and decrypts it.
	DecryptOAuthToken(encryptedToken EncryptedData) (oauth2.Token, error)
	// EncryptString encrypts a string.
	EncryptString(data string) (EncryptedData, error)
	// DecryptString decrypts a string encrypted with EncryptString.
	DecryptString(encryptedString EncryptedData) (string, error)
}

var (
	// ErrDecrypt is returned when we cannot decrypt a secret.
	ErrDecrypt = errors.New("unable to decrypt")
	// ErrEncrypt is returned when we cannot encrypt a secret.
	ErrEncrypt = errors.New("unable to encrypt")
)

type algorithmsByName map[algorithms.Type]algorithms.EncryptionAlgorithm

type engine struct {
	keystore            keystores.KeyStore
	supportedAlgorithms algorithmsByName
	defaultKeyID        string
	defaultAlgorithm    algorithms.Type
}

const (
	// DefaultAlgorithm defines the default algorithm to use for encryption.
	DefaultAlgorithm = algorithms.Aes256Gcm
	// FallbackAlgorithm defines an older algorithm we use for old data.
	FallbackAlgorithm = algorithms.Aes256Cfb
)

// NewEngineFromConfig creates a new crypto engine from the service config
func NewEngineFromConfig(config *serverconfig.Config) (Engine, error) {
	// Use fallback if the new config structure is missing
	var cryptoCfg serverconfig.CryptoConfig
	if config.Crypto.Default.KeyID != "" {
		cryptoCfg = config.Crypto
	} else if config.Auth.TokenKey != "" {
		fallbackConfig, err := convertToCryptoConfig(&config.Auth)
		if err != nil {
			return nil, fmt.Errorf("unable to load fallback config: %w", err)
		}
		cryptoCfg = fallbackConfig
	} else {
		return nil, errors.New("no encryption keys configured")
	}

	keystore, err := keystores.NewKeyStoreFromConfig(cryptoCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create keystore: %w", err)
	}

	// Ensure critical defaults are always loaded, plus any defined in the config.
	algoTypes := []algorithms.Type{
		DefaultAlgorithm,
		FallbackAlgorithm,
	}

	for _, algoStr := range cryptoCfg.SupportedAlgorithms {
		algoTypes = append(algoTypes, algorithms.Type(algoStr))
	}

	// Assuming cryptoCfg.SupportedAlgorithms is added to serverconfig.
	// If not, this loop safely defaults to the base types without breaking.
	// algoTypes = append(algoTypes, cryptoCfg.SupportedAlgorithms...)
	supportedAlgorithms := make(algorithmsByName)
	for _, algoType := range algoTypes {
		// Prevent duplicate instantiation
		if _, exists := supportedAlgorithms[algoType]; exists {
			continue
		}

		algorithm, err := algorithms.NewFromType(algoType)
		if err != nil {
			return nil, fmt.Errorf("failed to instantiate algorithm %s: %w", algoType, err)
		}
		supportedAlgorithms[algoType] = algorithm
	}

	return &engine{
		keystore:            keystore,
		supportedAlgorithms: supportedAlgorithms,
		defaultAlgorithm:    DefaultAlgorithm,
		defaultKeyID:        cryptoCfg.Default.KeyID,
	}, nil
}

func (e *engine) EncryptOAuthToken(token *oauth2.Token) (EncryptedData, error) {
	// Convert token to JSON.
	//nolint:gosec // We're about to encrypt jsonData. It shouldn't be used beyond this function.
	jsonData, err := json.Marshal(token)
	if err != nil {
		return EncryptedData{}, fmt.Errorf("unable to marshal token to json: %w", err)
	}

	// Encrypt the JSON.
	encrypted, err := e.encrypt(jsonData)
	if err != nil {
		return EncryptedData{}, err
	}
	return encrypted, nil
}

func (e *engine) DecryptOAuthToken(encryptedToken EncryptedData) (result oauth2.Token, err error) {
	// Decrypt the token.
	token, err := e.decrypt(encryptedToken)
	if err != nil {
		return result, err
	}

	// Deserialize to token struct.
	err = json.Unmarshal(token, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (e *engine) EncryptString(data string) (EncryptedData, error) {
	encrypted, err := e.encrypt([]byte(data))
	if err != nil {
		return EncryptedData{}, err
	}
	return encrypted, nil
}

func (e *engine) DecryptString(encryptedString EncryptedData) (string, error) {
	decrypted, err := e.decrypt(encryptedString)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

func (e *engine) encrypt(plaintext []byte) (EncryptedData, error) {
	// Neither of these lookups should ever fail.
	algorithm, ok := e.supportedAlgorithms[e.defaultAlgorithm]
	if !ok {
		return EncryptedData{}, fmt.Errorf("unable to find preferred algorithm: %s", e.defaultAlgorithm)
	}

	key, err := e.keystore.GetKey(e.defaultKeyID)
	if err != nil {
		return EncryptedData{}, fmt.Errorf("unable to find preferred key with ID: %s", e.defaultKeyID)
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return EncryptedData{}, errors.Join(ErrEncrypt, fmt.Errorf("failed to generate salt: %w", err))
	}

	// Pass the salt into the encryption algorithm
	encrypted, err := algorithm.Encrypt(plaintext, key, salt)
	if err != nil {
		return EncryptedData{}, errors.Join(ErrEncrypt, err)
	}

	return EncryptedData{
		Algorithm:   e.defaultAlgorithm,
		EncodedData: base64.StdEncoding.EncodeToString(encrypted),
		KeyVersion:  e.defaultKeyID,
		Salt:        base64.StdEncoding.EncodeToString(salt),
	}, nil
}

func (e *engine) decrypt(data EncryptedData) ([]byte, error) {
	if data.EncodedData == "" {
		return nil, errors.New("cannot decrypt empty data")
	}

	algorithm, ok := e.supportedAlgorithms[data.Algorithm]
	if !ok {
		return nil, fmt.Errorf("%w: %s", algorithms.ErrUnknownAlgorithm, data.Algorithm)
	}

	key, err := e.keystore.GetKey(data.KeyVersion)
	if err != nil {
		// error from keystore is good enough - we do not need more context
		return nil, err
	}

	// base64 decode the string
	encrypted, err := base64.StdEncoding.DecodeString(data.EncodedData)
	if err != nil {
		return nil, fmt.Errorf("error decoding secret: %w", err)
	}

	// Decode the salt if it exists (legacy data will have an empty salt string)
	var salt []byte
	if data.Salt != "" {
		salt, err = base64.StdEncoding.DecodeString(data.Salt)
		if err != nil {
			return nil, fmt.Errorf("error decoding salt: %w", err)
		}
	}

	result, err := algorithm.Decrypt(encrypted, key, salt)
	if err != nil {
		return nil, errors.Join(ErrDecrypt, err)
	}

	return result, nil
}

// This is for config transition purposes, and will eventually be removed.
func convertToCryptoConfig(a *serverconfig.AuthConfig) (serverconfig.CryptoConfig, error) {
	abspath, err := filepath.Abs(a.TokenKey)
	if err != nil {
		return serverconfig.CryptoConfig{}, fmt.Errorf("could not get absolute path: %w", err)
	}
	name := filepath.Base(abspath)
	dir := filepath.Dir(abspath)

	return serverconfig.CryptoConfig{
		KeyStore: serverconfig.KeyStoreConfig{
			Type: keystores.LocalKeyStore,
			Local: serverconfig.LocalKeyStoreConfig{
				KeyDir: dir,
			},
		},
		Default: serverconfig.DefaultCrypto{
			KeyID: name,
		},
	}, nil
}
