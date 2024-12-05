// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package crypto_test

import (
	"testing"

	"github.com/mindersec/minder/internal/crypto"
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
