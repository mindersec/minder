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

	"golang.org/x/oauth2"

	serverconfig "github.com/stacklok/minder/internal/config/server"
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

type engine struct {
	algorithm EncryptionAlgorithm
}

// NewEngineFromAuthConfig creates a new crypto engine from an auth config
func NewEngineFromAuthConfig(authConfig *serverconfig.AuthConfig) (Engine, error) {
	if authConfig == nil {
		return nil, errors.New("auth config is nil")
	}

	keyBytes, err := authConfig.GetTokenKey()
	if err != nil {
		return nil, fmt.Errorf("failed to read token key file: %s", err)
	}

	return NewEngine(keyBytes), nil
}

// NewEngine creates the engine based on the specified algorithm and key.
func NewEngine(key []byte) Engine {
	return &engine{algorithm: newAlgorithm(key)}
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
		return EncryptedData{}, fmt.Errorf("unable to encrypt token: %w", err)
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
		return "", fmt.Errorf("%w: %w", ErrDecrypt, err)
	}
	return string(decrypted), nil
}

func (e *engine) encrypt(data []byte) (EncryptedData, error) {
	encrypted, err := e.algorithm.Encrypt(data, legacySalt)
	if err != nil {
		return EncryptedData{}, err
	}

	encoded := base64.StdEncoding.EncodeToString(encrypted)
	// TODO:
	// 1. when we support more than one algorithm, remove hard-coding.
	// 2. Allow salt to be randomly generated per secret.
	// 3. Set key version.
	return NewBackwardsCompatibleEncryptedData(encoded), nil
}

func (e *engine) decrypt(data EncryptedData) ([]byte, error) {
	// TODO: Select algorithm based on Algorithm field when we support
	// more than one algorithm.
	if data.Algorithm != Aes256Cfb {
		return nil, fmt.Errorf("%w: %s", ErrUnknownAlgorithm, data.Algorithm)
	}

	// base64 decode the string
	encrypted, err := base64.StdEncoding.DecodeString(data.EncodedData)
	if err != nil {
		return nil, err
	}

	// decrypt the data
	return e.algorithm.Decrypt(encrypted, data.Salt)
}
