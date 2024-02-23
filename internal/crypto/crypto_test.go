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

package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEncryptDecryptBytes(t *testing.T) {
	t.Parallel()

	encrypted, err := EncryptBytes("test", []byte("test"))
	assert.Nil(t, err)
	decrypted, err := decryptBytes("test", encrypted)
	assert.Nil(t, err)
	assert.Equal(t, "test", string(decrypted))
}

func TestEncryptTooLarge(t *testing.T) {
	t.Parallel()

	large := make([]byte, 34000000) // More than 32 MB
	_, err := EncryptBytes("test", large)
	assert.ErrorIs(t, err, status.Error(codes.InvalidArgument, "data is too large (>32MB)"))
}

func TestGenerateNonce(t *testing.T) {
	t.Parallel()

	state, err := GenerateNonce()
	if err != nil {
		t.Errorf("Error in generateState: %v", err)
	}

	if len(state) != 54 {
		t.Errorf("Unexpected length of state: %v", len(state))
	}
}

func TestIsNonceValid(t *testing.T) {
	t.Parallel()

	nonce, err := GenerateNonce()
	if err != nil {
		t.Errorf("Error in generateState: %v", err)
	}

	valid, err := IsNonceValid(nonce, 3600)
	if err != nil {
		t.Errorf("Error in isNonceValid: %v", err)
	}

	if !valid {
		t.Errorf("Expected nonce to be valid, got invalid")
	}

	invalid := "AAAAAGSDmJ_tKMkuUmeoOBdSQGWXq3BE_Zp7IrUFVUau5HcPa-yvzQ"

	valid, err = IsNonceValid(invalid, 3600)
	if err != nil {
		t.Errorf("Error in isNonceValid: %v", err)
	}

	if valid {
		t.Errorf("Expected nonce to be invalid, got valid")
	}
}

func TestEncryptDecryptString(t *testing.T) {
	t.Parallel()

	engine := Engine{
		encryptionKey: "test",
	}

	originalString := "testString"
	encrypted, err := engine.EncryptString(originalString)
	assert.Nil(t, err)
	decrypted, err := engine.DecryptString(encrypted)
	assert.Nil(t, err)
	assert.Equal(t, originalString, decrypted)
}
