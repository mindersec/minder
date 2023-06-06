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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// GenerateToken generates a JWT token
func GenerateToken(userId int32, username string, accessPrivateKey []byte, refreshPrivateKey []byte,
	expiry int64, refreshExpiry int64) (string, string, int64, int64, error) {
	if accessPrivateKey == nil || refreshPrivateKey == nil {
		return "", "", 0, 0, fmt.Errorf("invalid key")
	}
	tokenExpirationTime := time.Now().Add(time.Duration(expiry) * time.Minute).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"userId":   userId,
		"username": username,
		"iat":      time.Now().Unix(),
		"exp":      tokenExpirationTime,
	})

	accessKey, err := jwt.ParseRSAPrivateKeyFromPEM(accessPrivateKey)
	if err != nil {
		return "", "", 0, 0, err
	}
	tokenString, err := token.SignedString(accessKey)
	if err != nil {
		return "", "", 0, 0, err
	}

	// Create a refresh token that lasts longer than the access token
	refreshExpirationTime := time.Now().Add(time.Duration(refreshExpiry) * time.Minute).Unix()

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"userId":   userId,
		"username": username,
		"iat":      time.Now().Unix(),
		"exp":      refreshExpirationTime,
	})

	refreshKey, err := jwt.ParseRSAPrivateKeyFromPEM(refreshPrivateKey)
	if err != nil {
		return "", "", 0, 0, err
	}

	refreshTokenString, err := refreshToken.SignedString(refreshKey)
	if err != nil {
		return "", "", 0, 0, err
	}

	return tokenString, refreshTokenString, tokenExpirationTime, refreshExpirationTime, nil
}

// VerifyToken verifies the token string and returns the user ID
func VerifyToken(tokenString string, publicKey []byte) (uint, string, error) {
	// extract the pubkey from the pem
	pubPem, _ := pem.Decode(publicKey)
	if pubPem == nil {
		return 0, "", fmt.Errorf("invalid key")
	}
	key, err := x509.ParsePKCS1PublicKey(pubPem.Bytes)
	if err != nil {
		return 0, "", fmt.Errorf("invalid key")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	})

	if err != nil {
		return 0, "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, "", fmt.Errorf("invalid token")
	}

	userIdFloat, ok := claims["userId"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("invalid user ID format")
	}
	userId := uint(userIdFloat)

	userName, ok := claims["username"].(string)
	if !ok {
		return 0, "", fmt.Errorf("invalid username format")
	}

	return userId, userName, nil
}
