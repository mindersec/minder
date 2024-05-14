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

// Package crypto provides cryptographic functions
package crypto

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/oauth2"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto/algorithms"
	"github.com/stacklok/minder/internal/crypto/keystores"
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
	// TODO: get rid of this when we allow per-secret salting.
	legacySalt = []byte("somesalt")
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

// NewEngineFromAuthConfig creates a new crypto engine from the service config
// TODO: modify to support multiple keys/algorithms
func NewEngineFromAuthConfig(config *serverconfig.AuthConfig) (Engine, error) {
	if config == nil {
		return nil, errors.New("auth config is nil")
	}

	keystore, err := keystores.NewKeyStoreFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to read token key file: %s", err)
	}

	aes, err := algorithms.NewFromType(algorithms.Aes256Cfb)
	if err != nil {
		return nil, err
	}
	supportedAlgorithms := map[algorithms.Type]algorithms.EncryptionAlgorithm{
		algorithms.Aes256Cfb: aes,
	}

	return &engine{
		keystore:            keystore,
		supportedAlgorithms: supportedAlgorithms,
		defaultAlgorithm:    algorithms.Aes256Cfb,
		// Use the key filename as the key ID.
		// This will be cleaned up in a future PR
		// Right now, by the time we get here, this should return a valid result
		defaultKeyID: filepath.Base(config.TokenKey),
	}, nil
}

func (e *engine) EncryptOAuthToken(token *oauth2.Token) (EncryptedData, error) {
	// Convert token to JSON.
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

func (e *engine) encrypt(data []byte) (EncryptedData, error) {
	// Neither of these lookups should ever fail.
	algorithm, ok := e.supportedAlgorithms[e.defaultAlgorithm]
	if !ok {
		return EncryptedData{}, fmt.Errorf("unable to find preferred algorithm: %s", e.defaultAlgorithm)
	}

	key, err := e.keystore.GetKey(e.defaultAlgorithm, e.defaultKeyID)
	if err != nil {
		return EncryptedData{}, fmt.Errorf("unable to find preferred key with ID: %s", e.defaultKeyID)
	}

	encrypted, err := algorithm.Encrypt(data, key, legacySalt)
	if err != nil {
		return EncryptedData{}, errors.Join(ErrEncrypt, err)
	}

	encoded := base64.StdEncoding.EncodeToString(encrypted)
	// TODO: Allow salt to be randomly generated per secret.
	return EncryptedData{
		Algorithm:   e.defaultAlgorithm,
		EncodedData: encoded,
		Salt:        legacySalt,
		KeyVersion:  e.defaultKeyID,
	}, nil
}

func (e *engine) decrypt(data EncryptedData) ([]byte, error) {
	algorithm, ok := e.supportedAlgorithms[data.Algorithm]
	if !ok {
		return nil, fmt.Errorf("%w: %s", algorithms.ErrUnknownAlgorithm, e.defaultAlgorithm)
	}

	key, err := e.keystore.GetKey(e.defaultAlgorithm, e.defaultKeyID)
	if err != nil {
		// error from keystore is good enough - we do not need more context
		return nil, err
	}

	// base64 decode the string
	encrypted, err := base64.StdEncoding.DecodeString(data.EncodedData)
	if err != nil {
		return nil, fmt.Errorf("error decoding secret: %w", err)
	}

	// decrypt the data
	result, err := algorithm.Decrypt(encrypted, key, data.Salt)
	if err != nil {
		return nil, errors.Join(ErrDecrypt, err)
	}
	return result, nil
}
