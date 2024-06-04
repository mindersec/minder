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
			Name:          "GCM Encrypt rejects key which cannot be base64 decoded",
			Key:           []byte{0xFF, 0xFF, 0xFF, 0xFF},
			Plaintext:     []byte(plaintext),
			ExpectedError: "unable to base64 decode the encryption key",
		},
		{
			Name:          "GCM Encrypt rejects short key",
			Key:           []byte{0x41, 0x42, 0x43, 0x44},
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
			Name:          "GCM Decrypt rejects key which cannot be base64 decoded",
			Key:           []byte{0xFF},
			Ciphertext:    []byte(plaintext),
			ExpectedError: "unable to base64 decode the encryption key",
		},
		{
			Name:          "GCM Decrypt rejects short key",
			Key:           []byte{0xa},
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
			Name:          "GCM Decrypt rejects undersized ciphertext",
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
	key = []byte("2hcGLimy2i7LAknby2AFqYx87CaaCAtjxDiorRxYq8Q=")
	gcm = algorithms.AES256GCMAlgorithm{}
)

const plaintext = "Hello world"
