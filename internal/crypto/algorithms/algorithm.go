// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	// Aes256Gcm is the AES-256-GCM algorithm
	Aes256Gcm Type = "aes-256-gcm"
)

const maxPlaintextSize = 32 * 1024 * 1024

var (
	// ErrUnknownAlgorithm is returned when an incorrect algorithm name is used.
	ErrUnknownAlgorithm = errors.New("unexpected encryption algorithm")
	// ErrExceedsMaxSize is returned when the plaintext is too large.
	ErrExceedsMaxSize = errors.New("plaintext is too large, limited to 32MiB")
)

// TypeFromString attempts to map a string to a `Type` value.
func TypeFromString(name string) (Type, error) {
	// TODO: use switch when we support more than once type.
	switch name {
	case string(Aes256Cfb):
		return Aes256Cfb, nil
	case string(Aes256Gcm):
		return Aes256Gcm, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownAlgorithm, name)
	}
}

// NewFromType instantiates an encryption algorithm by name
func NewFromType(algoType Type) (EncryptionAlgorithm, error) {
	switch algoType {
	case Aes256Cfb:
		return &AES256CFBAlgorithm{}, nil
	case Aes256Gcm:
		return &AES256GCMAlgorithm{}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownAlgorithm, algoType)
	}
}
