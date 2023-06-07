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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// UserClaims contains the claims for a user
type UserClaims struct {
	UserId         int32
	RoleId         int32
	GroupId        int32
	OrganizationId int32
	IsAdmin        bool
	IsSuperadmin   bool
}

// GenerateToken generates a JWT token
func GenerateToken(userClaims UserClaims, accessPrivateKey []byte, refreshPrivateKey []byte,
	expiry int64, refreshExpiry int64) (string, string, int64, int64, error) {
	if accessPrivateKey == nil || refreshPrivateKey == nil {
		return "", "", 0, 0, fmt.Errorf("invalid key")
	}
	tokenExpirationTime := time.Now().Add(time.Duration(expiry) * time.Second).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"userId":  userClaims.UserId,
		"roleId":  userClaims.RoleId,
		"groupId": userClaims.GroupId,
		"orgId":   userClaims.OrganizationId,
		"isAdmin": userClaims.IsAdmin,
		"isSuper": userClaims.IsSuperadmin,
		"iat":     time.Now().Unix(),
		"exp":     tokenExpirationTime,
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
	refreshExpirationTime := time.Now().Add(time.Duration(refreshExpiry) * time.Second).Unix()

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"userId": userClaims.UserId,
		"iat":    time.Now().Unix(),
		"exp":    refreshExpirationTime,
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
func VerifyToken(tokenString string, publicKey []byte) (UserClaims, error) {
	var userClaims UserClaims
	// extract the pubkey from the pem
	pubPem, _ := pem.Decode(publicKey)
	if pubPem == nil {
		return userClaims, fmt.Errorf("invalid key")
	}
	key, err := x509.ParsePKCS1PublicKey(pubPem.Bytes)
	if err != nil {
		// try another method
		key1, err := x509.ParsePKIXPublicKey(pubPem.Bytes)
		if err != nil {
			return userClaims, fmt.Errorf("invalid key")
		}
		key = key1.(*rsa.PublicKey)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	})

	if err != nil {
		return userClaims, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return userClaims, fmt.Errorf("invalid token")
	}

	// generate claims
	userClaims.UserId = claims["userId"].(int32)
	userClaims.RoleId = claims["roleId"].(int32)
	userClaims.GroupId = claims["groupId"].(int32)
	userClaims.OrganizationId = claims["orgId"].(int32)
	userClaims.IsAdmin = claims["isAdmin"].(bool)
	userClaims.IsSuperadmin = claims["isSuper"].(bool)

	return userClaims, nil
}
