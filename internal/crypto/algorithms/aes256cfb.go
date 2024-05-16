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

package algorithms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AES256CFBAlgorithm implements the AES-256-CFB algorithm
type AES256CFBAlgorithm struct{}

// Encrypt encrypts a row of data.
func (a *AES256CFBAlgorithm) Encrypt(data []byte, key []byte, salt []byte) ([]byte, error) {
	if len(data) > maxSize {
		return nil, status.Errorf(codes.InvalidArgument, "data is too large (>32MB)")
	}
	block, err := aes.NewCipher(a.deriveKey(key, salt))
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
func (a *AES256CFBAlgorithm) Decrypt(ciphertext []byte, key []byte, salt []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.deriveKey(key, salt))
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create cipher: %s", err)
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
func (_ *AES256CFBAlgorithm) deriveKey(key []byte, salt []byte) []byte {
	return argon2.IDKey(key, salt, 1, 64*1024, 4, 32)
}
