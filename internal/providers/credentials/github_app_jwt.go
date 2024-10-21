//
// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"crypto/rsa"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// CreateGitHubAppJWT creates a JWT token for a GitHub App
func CreateGitHubAppJWT(appId int64, privateKey *rsa.PrivateKey) (string, error) {
	// Create the Claims
	claims := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 10).Unix(),
		"iss": strconv.FormatInt(appId, 10),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	jwtToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("unable to sign JWT token: %v", err)
	}

	return jwtToken, nil
}
