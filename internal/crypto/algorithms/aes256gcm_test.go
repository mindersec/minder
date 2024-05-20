// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package algorithms_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/crypto/algorithms"
)

func TestGCMEncrypt(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		Key           []byte
		Plaintext     []byte
		ExpectedError string
	}{
		{
			Name:          "GCM Encrypt rejects short key",
			Key:           []byte{0xFF},
			Plaintext:     []byte(plaintext),
			ExpectedError: "invalid key size",
		},
		{
			Name:          "GCM Encrypt rejects oversized plaintext",
			Key:           key,
			Plaintext:     make([]byte, 33*1024*1024), // 33MiB
			ExpectedError: algorithms.ErrExceedsMaxSize.Error(),
		},
		{
			Name:      "GCM encrypts plaintext",
			Key:       key,
			Plaintext: []byte(plaintext),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			result, err := gcm.Encrypt(scenario.Plaintext, scenario.Key)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
				// validate by decrypting
				decrypted, err := gcm.Decrypt(result, key)
				require.NoError(t, err)
				require.Equal(t, scenario.Plaintext, decrypted)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

// This doesn't test decryption - that is tested in the happy path of the encrypt test
func TestGCMDecrypt(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		Key           []byte
		Ciphertext    []byte
		ExpectedError string
	}{
		{
			Name:          "GCM Decrypt rejects short key",
			Key:           []byte{0xFF},
			Ciphertext:    []byte(plaintext),
			ExpectedError: "invalid key size",
		},
		{
			Name:          "GCM Decrypt rejects malformed ciphertext",
			Key:           key,
			Ciphertext:    make([]byte, 32), // 33MiB
			ExpectedError: "message authentication failed",
		},
		{
			Name:          "GCM Decrypt rejects undersized key",
			Key:           key,
			Ciphertext:    []byte{0xFF},
			ExpectedError: "malformed ciphertext",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			_, err := gcm.Decrypt(scenario.Ciphertext, scenario.Key)
			require.ErrorContains(t, err, scenario.ExpectedError)
		})
	}
}

var (
	key = []byte{
		0x5, 0x94, 0x74, 0xfd, 0xb7, 0xf9, 0x85, 0x9, 0x67, 0x8, 0x2D, 0xe8, 0x46, 0x8c, 0x76, 0xe2,
		0x7a, 0x85, 0x7f, 0xed, 0x67, 0xd4, 0xd5, 0x2c, 0x46, 0x00, 0xba, 0x44, 0x8d, 0x54, 0x20, 0xf1,
	}
	gcm = algorithms.AES256GCMAlgorithm{}
)

const plaintext = "Hello world"
