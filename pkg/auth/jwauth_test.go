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

// package auth contains the authentication logic for the control plane
package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/db"
)

const (
	testUserID     = 123
	testUsername   = "test_user"
	testExpiry     = 10   // 10 minutes
	testRefreshExp = 1440 // 24 hours in minutes
)

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	// generate test keys
	access_key, _ := util.RandomKeypair(2048)
	refresh_key, _ := util.RandomKeypair(2048)

	// Test valid token generation
	claims := UserClaims{
		UserId: testUserID,
	}
	tokenString, refreshTokenString, tokenExp, refreshTokenExp, err := GenerateToken(claims, access_key, refresh_key, testExpiry, testRefreshExp)
	if err != nil {
		t.Errorf("Error generating token: %v", err)
	}
	if len(tokenString) == 0 || len(refreshTokenString) == 0 {
		t.Error("Token or refresh token string is empty")
	}
	if tokenExp <= time.Now().Unix() || refreshTokenExp <= time.Now().Unix() {
		t.Error("Token or refresh token has already expired")
	}
	if refreshTokenExp <= tokenExp {
		t.Error("Refresh token expires before access token")
	}

	// Test error case with invalid key
	_, _, _, _, err = GenerateToken(claims, nil, nil, testExpiry, testRefreshExp)
	if err == nil {
		t.Error("Expected error with invalid key, but got nil")
	} else if err.Error() != "invalid key" {
		t.Errorf("Expected error message 'invalid key', but got '%s'", err.Error())
	}
}

func TestVerifyToken(t *testing.T) {
	t.Parallel()

	claims := UserClaims{
		UserId: testUserID,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mockdb.NewMockStore(ctrl)

	// generate test keys
	access_key, access_pub_key := util.RandomKeypair(2048)
	refresh_key, _ := util.RandomKeypair(2048)

	// Test valid token verification
	tokenString, _, _, _, err := GenerateToken(claims, access_key, refresh_key, testExpiry, testRefreshExp)
	if err != nil {
		t.Errorf("Error generating token: %v", err)
	}

	mockStore.EXPECT().GetUserByID(context.Background(), gomock.Any())

	retClaims, err := VerifyToken(tokenString, access_pub_key, mockStore)
	if err != nil {
		t.Errorf("Error verifying token: %v", err)
	}

	if retClaims.UserId != testUserID {
		t.Errorf("Expected user ID %d, but got %d", testUserID, retClaims.UserId)
	}

	// Test error case with invalid token string
	_, err = VerifyToken("invalid_token_string", access_key, mockStore)
	if err == nil {
		t.Error("Expected error with invalid token string, but got nil")
	}

	// Test error case with invalid key
	_, err = VerifyToken(tokenString, []byte("invalid_key"), mockStore)
	if err == nil {
		t.Error("Expected error with invalid key, but got nil")
	} else {
		t.Logf("Successfully received error when using an invalid key: %v", err)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	t.Parallel()

	claims := UserClaims{
		UserId: testUserID,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mockdb.NewMockStore(ctrl)

	// generate test keys
	access_key, access_pub_key := util.RandomKeypair(2048)
	refresh_key, _ := util.RandomKeypair(2048)

	tokenString, _, _, _, err := GenerateToken(claims, access_key, refresh_key, testExpiry, testRefreshExp)
	if err != nil {
		t.Errorf("Error generating token: %v", err)
	}

	currentTime := time.Unix(time.Now().Unix()+3600, 0)
	mockStore.EXPECT().GetUserByID(context.Background(),
		gomock.Any()).Return(db.User{MinTokenIssuedTime: sql.NullTime{Time: currentTime, Valid: true}}, nil)

	_, err = VerifyToken(tokenString, access_pub_key, mockStore)
	if err != nil {
		t.Logf("Successfully received error when using an expired token: %v", err)
	}
}
