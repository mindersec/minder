// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Adapted from: https://github.com/gtank/cryptopasta/blob/bc3a108a5776376aa811eea34b93383837994340/encrypt.go
// cryptopasta - basic cryptography examples
//
// Written in 2015 by George Tankersley <george.tankersley@gmail.com>
//
// To the extent possible under law, the author(s) have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
//
// You should have received a copy of the CC0 Public Domain Dedication along
// with this software. If not, see // <http://creativecommons.org/publicdomain/zero/1.0/>.

package algorithms

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// AES256GCMAlgorithm provides symmetric authenticated encryption using 256-bit AES-GCM.
type AES256GCMAlgorithm struct{}

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func (*AES256GCMAlgorithm) Encrypt(plaintext []byte, key []byte, salt []byte) ([]byte, error) {
	if len(plaintext) > maxPlaintextSize {
		return nil, ErrExceedsMaxSize
	}

	decodedKey, err := decodeKey(key)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(salt) < nonceSize {
		return nil, fmt.Errorf("provided salt is too short for GCM nonce (need %d bytes)", nonceSize)
	}
	nonce := salt[:nonceSize]

	// The salt is already being saved in the database by engine.go.
	// Seal prepends the nonce to the ciphertext
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func (*AES256GCMAlgorithm) Decrypt(ciphertext []byte, key []byte, salt []byte) ([]byte, error) {
	decodedKey, err := decodeKey(key)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(salt) < nonceSize {
		return nil, fmt.Errorf("provided salt is too short for GCM decryption (need %d bytes)", nonceSize)
	}
	nonce := salt[:nonceSize]

	if len(ciphertext) < gcm.Overhead() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

func decodeKey(key []byte) ([]byte, error) {
	cleanKey := strings.TrimSpace(string(key))

	decodedKey, err := base64.StdEncoding.DecodeString(cleanKey)
	if err != nil {
		return nil, fmt.Errorf("unable to base64 decode the encryption key: %w", err)
	}

	// FIX: Safety check to ensure AES-256 gets exactly 32 bytes
	if len(decodedKey) != 32 {
		return nil, fmt.Errorf("invalid key size: expected 32 bytes, got %d", len(decodedKey))
	}

	return decodedKey, nil
}
