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

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EncryptionAlgorithm represents a crypto algorithm used by the Engine
type EncryptionAlgorithm interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

const maxSize = 32 * 1024 * 1024

// In a real application, you should use a unique salt for
// each key and save it with the encrypted data.
var (
	salt                = []byte("somesalt")
	errUnknownAlgorithm = errors.New("unexpected encryption algorithm")
)

// EncryptionAlgorithmType is an enum of supported encryption algorithms
type EncryptionAlgorithmType string

const (
	// AESCFB is the AES-CFB algorithm
	AESCFB EncryptionAlgorithmType = "aes-cfb"
)

// AlgorithmTypeFromString converts a string to an EncryptionAlgorithmType
// or returns errUnknownAlgorithm.
func AlgorithmTypeFromString(input string) (EncryptionAlgorithmType, error) {
	// for backwards compatibility - default to AES-CFB if string is empty
	if input == "" || input == string(AESCFB) {
		return AESCFB, nil
	}
	return "", fmt.Errorf("%w: %s", errUnknownAlgorithm, input)
}

func newAlgorithm(key []byte) EncryptionAlgorithm {
	// TODO: Make the type of algorithm selectable
	return &aesCFBSAlgorithm{encryptionKey: key}
}

type aesCFBSAlgorithm struct {
	encryptionKey []byte
}

// Encrypt encrypts a row of data.
func (a *aesCFBSAlgorithm) Encrypt(data []byte) ([]byte, error) {
	if len(data) > maxSize {
		return nil, status.Errorf(codes.InvalidArgument, "data is too large (>32MB)")
	}
	block, err := aes.NewCipher(a.deriveKey())
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create cipher: %s", err)
	}

	// The IV needs to be unique, but not secure. Therefore, it's common to include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to read random bytes: %s", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	return ciphertext, nil
}

// Decrypt decrypts a row of data.
func (a *aesCFBSAlgorithm) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.deriveKey())
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create cipher: %s", err)
	}

	// The IV needs to be extracted from the ciphertext.
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

// Function to derive a key from a passphrase using Argon2
func (a *aesCFBSAlgorithm) deriveKey() []byte {
	return argon2.IDKey(a.encryptionKey, salt, 1, 64*1024, 4, 32)
}
