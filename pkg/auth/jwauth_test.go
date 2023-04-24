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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {

	tokenString, tokenExpirationTime, err := GenerateToken(123, "secret", 60)

	// Check that token generation succeeds and returns a non-empty token string
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if tokenString == "" {
		t.Errorf("Expected a non-empty token string")
	}

	// Check that token expiration time is 60 minutes from now
	expectedExpirationTime := time.Now().Add(60 * time.Minute).Unix()

	t.Log("expectedExpirationTime", expectedExpirationTime)
	if tokenExpirationTime != expectedExpirationTime {
		t.Errorf("Expected token expiration time to be %d, but got %d", expectedExpirationTime, tokenExpirationTime)
	}
}

func TestVerifyToken(t *testing.T) {
	// Generate a token for testing
	tokenString, _, err := GenerateToken(123, "secret", 60)
	if err != nil {
		t.Errorf("Unexpected error generating token: %v", err)
	}

	// Check that token verification succeeds and returns the correct user ID
	userId, err := VerifyToken(tokenString, "secret")
	if err != nil {
		t.Errorf("Unexpected error verifying token: %v", err)
	}

	if userId != 123 {
		t.Errorf("Expected user ID to be %d, but got %d", 123, userId)
	}

	// Check that token verification fails with an invalid token
	_, err = VerifyToken("invalid_token", "secret")
	if err == nil {
		t.Errorf("Expected token verification to fail with an invalid token")
	}
}
