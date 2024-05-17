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

// Package algorithms contains implementations of various crypto algorithms
// for the crypto engine.
package algorithms

import (
	"errors"
	"fmt"
)

// EncryptionAlgorithm represents a crypto algorithm used by the Engine
type EncryptionAlgorithm interface {
	Encrypt(plaintext []byte, key []byte) ([]byte, error)
	Decrypt(ciphertext []byte, key []byte) ([]byte, error)
}

// Type is an enum of supported encryption algorithms
type Type string

const (
	// Aes256Cfb is the AES-256-CFB algorithm
	Aes256Cfb Type = "aes-256-cfb"
)

const maxSize = 32 * 1024 * 1024

// ErrUnknownAlgorithm is used when an incorrect algorithm name is used.
var (
	ErrUnknownAlgorithm = errors.New("unexpected encryption algorithm")
)

// TypeFromString attempts to map a string to a `Type` value.
func TypeFromString(name string) (Type, error) {
	// TODO: use switch when we support more than once type.
	if name == string(Aes256Cfb) {
		return Aes256Cfb, nil
	}
	return "", fmt.Errorf("%w: %s", ErrUnknownAlgorithm, name)
}

// NewFromType instantiates an encryption algorithm by name
func NewFromType(algoType Type) (EncryptionAlgorithm, error) {
	// TODO: use switch when we support more than once type.
	if algoType == Aes256Cfb {
		return &AES256CFBAlgorithm{}, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownAlgorithm, algoType)
}
