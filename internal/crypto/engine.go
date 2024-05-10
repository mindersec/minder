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
	EncryptOAuthToken(token *oauth2.Token) (string, error)
	DecryptOAuthToken(encToken string) (oauth2.Token, error)
	EncryptString(data string) (string, error)
	DecryptString(encData string) (string, error)
}

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

// EncryptOAuthToken encrypts an oauth token
func (e *engine) EncryptOAuthToken(token *oauth2.Token) (string, error) {
	// Convert token to JSON
	jsonData, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("unable to marshal token to json: %w", err)
	}
	encrypted, err := e.algorithm.Encrypt(jsonData)
	if err != nil {
		return "", fmt.Errorf("unable to encrypt token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptOAuthToken decrypts an encrypted oauth token
func (e *engine) DecryptOAuthToken(encToken string) (oauth2.Token, error) {
	var decryptedToken oauth2.Token

	// base64 decode the token
	decodeToken, err := base64.StdEncoding.DecodeString(encToken)
	if err != nil {
		return decryptedToken, err
	}

	// decrypt the token
	token, err := e.algorithm.Decrypt(decodeToken)
	if err != nil {
		return decryptedToken, err
	}

	// serialise token *oauth.Token
	err = json.Unmarshal(token, &decryptedToken)
	if err != nil {
		return decryptedToken, err
	}
	return decryptedToken, nil
}

// EncryptString encrypts a string
func (e *engine) EncryptString(data string) (string, error) {
	encrypted, err := e.algorithm.Encrypt([]byte(data))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString decrypts an encrypted string
func (e *engine) DecryptString(encData string) (string, error) {
	var decrypted string

	// base64 decode the string
	decodeToken, err := base64.StdEncoding.DecodeString(encData)
	if err != nil {
		return decrypted, err
	}

	// decrypt the string
	token, err := e.algorithm.Decrypt(decodeToken)
	if err != nil {
		return decrypted, err
	}

	return string(token), nil
}
