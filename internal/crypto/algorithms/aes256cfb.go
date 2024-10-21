// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package algorithms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// AES256CFBAlgorithm implements the AES-256-CFB algorithm
type AES256CFBAlgorithm struct{}

// Our current implementation of AES-256-CFB uses a fixed salt.
// Since we are planning to move to AES-256-GCM, leave this hardcoded here.
var legacySalt = []byte("somesalt")

// Encrypt encrypts a row of data.
func (a *AES256CFBAlgorithm) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(plaintext) > maxPlaintextSize {
		return nil, ErrExceedsMaxSize
	}
	block, err := aes.NewCipher(a.deriveKey(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// The IV needs to be unique, but not secure. Therefore, it's common to include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// Decrypt decrypts a row of data.
func (a *AES256CFBAlgorithm) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.deriveKey(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short to decrypt, length is: %d", len(ciphertext))
	}

	// The IV needs to be extracted from the ciphertext.
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

// Function to derive a key from a passphrase using Argon2
func (_ *AES256CFBAlgorithm) deriveKey(key []byte) []byte {
	return argon2.IDKey(key, legacySalt, 1, 64*1024, 4, 32)
}
