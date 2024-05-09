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

package crypto_test

import (
	"testing"

	"github.com/stacklok/minder/internal/crypto"
)

func TestGenerateNonce(t *testing.T) {
	t.Parallel()

	state, err := crypto.GenerateNonce()
	if err != nil {
		t.Errorf("Error in generateState: %v", err)
	}

	if len(state) != 54 {
		t.Errorf("Unexpected length of state: %v", len(state))
	}
}

func TestIsNonceValid(t *testing.T) {
	t.Parallel()

	nonce, err := crypto.GenerateNonce()
	if err != nil {
		t.Errorf("Error in generateState: %v", err)
	}

	valid, err := crypto.IsNonceValid(nonce, 3600)
	if err != nil {
		t.Errorf("Error in isNonceValid: %v", err)
	}

	if !valid {
		t.Errorf("Expected nonce to be valid, got invalid")
	}

	invalid := "AAAAAGSDmJ_tKMkuUmeoOBdSQGWXq3BE_Zp7IrUFVUau5HcPa-yvzQ"

	valid, err = crypto.IsNonceValid(invalid, 3600)
	if err != nil {
		t.Errorf("Error in isNonceValid: %v", err)
	}

	if valid {
		t.Errorf("Expected nonce to be invalid, got valid")
	}
}
