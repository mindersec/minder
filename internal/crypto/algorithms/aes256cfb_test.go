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

func TestCFBEncrypt(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		Key           []byte
		Plaintext     []byte
		ExpectedError string
	}{
		{
			Name:          "CFB Encrypt rejects oversized plaintext",
			Key:           key,
			Plaintext:     make([]byte, 33*1024*1024), // 33MiB
			ExpectedError: algorithms.ErrExceedsMaxSize.Error(),
		},
		{
			Name:      "CFB encrypts plaintext",
			Key:       key,
			Plaintext: []byte(plaintext),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			result, err := cfb.Encrypt(scenario.Plaintext, scenario.Key)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
				// validate by decrypting
				decrypted, err := cfb.Decrypt(result, key)
				require.NoError(t, err)
				require.Equal(t, scenario.Plaintext, decrypted)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

// This doesn't test decryption - that is tested in the happy path of the encrypt test
func TestCFBDecrypt(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		Key           []byte
		Ciphertext    []byte
		ExpectedError string
	}{
		{
			Name:          "CFB Decrypt rejects short key",
			Key:           []byte{0xFF},
			Ciphertext:    []byte(plaintext),
			ExpectedError: "ciphertext too short to decrypt",
		},
		{
			Name:          "CFB Decrypt rejects undersized ciphertext",
			Key:           key,
			Ciphertext:    []byte{0xFF},
			ExpectedError: "ciphertext too short to decrypt",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			_, err := cfb.Decrypt(scenario.Ciphertext, scenario.Key)
			require.ErrorContains(t, err, scenario.ExpectedError)
		})
	}
}

var (
	cfb = algorithms.AES256CFBAlgorithm{}
)
