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
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/mediator/internal/db"
)

// RoleInfo contains the role information for a user
type RoleInfo struct {
	RoleID         int32 `json:"role_id"`
	IsAdmin        bool  `json:"is_admin"`
	GroupID        int32 `json:"group_id"`
	OrganizationID int32 `json:"organization_id"`
}

// UserClaims contains the claims for a user
type UserClaims struct {
	UserId              int32
	GroupIds            []int32
	Roles               []RoleInfo
	OrganizationId      int32
	NeedsPasswordChange bool
}

// GenerateToken generates a JWT token
func GenerateToken(userClaims UserClaims, accessPrivateKey *rsa.PrivateKey, refreshPrivateKey *rsa.PrivateKey,
	expiry int64, refreshExpiry int64) (string, string, int64, int64, error) {
	if accessPrivateKey == nil || refreshPrivateKey == nil {
		return "", "", 0, 0, fmt.Errorf("invalid key")
	}
	tokenExpirationTime := time.Now().Add(time.Duration(expiry) * time.Second).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"userId":              int32(userClaims.UserId),
		"roleInfo":            userClaims.Roles,
		"groupIds":            userClaims.GroupIds,
		"orgId":               int32(userClaims.OrganizationId),
		"iat":                 time.Now().Unix(),
		"exp":                 tokenExpirationTime,
		"needsPasswordChange": userClaims.NeedsPasswordChange,
	})

	tokenString, err := token.SignedString(accessPrivateKey)
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

	refreshTokenString, err := refreshToken.SignedString(refreshPrivateKey)
	if err != nil {
		return "", "", 0, 0, err
	}

	return tokenString, refreshTokenString, tokenExpirationTime, refreshExpirationTime, nil
}

// VerifyToken verifies the token string and returns the user ID
// nolint:gocyclo
func VerifyToken(tokenString string, publicKey *rsa.PublicKey, store db.Store) (UserClaims, error) {
	var userClaims UserClaims
	if publicKey == nil {
		return userClaims, fmt.Errorf("invalid key")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, status.Error(codes.InvalidArgument, "unexpected signing method")
		}
		return publicKey, nil
	})

	if err != nil {
		return userClaims, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return userClaims, fmt.Errorf("invalid token")
	}

	// validate that iat is on the past
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if !claims.VerifyIssuedAt(time.Now().Unix(), true) {
			return userClaims, fmt.Errorf("token used before issued")
		}
	}

	// we have the user id, read the auth
	userId := int32(claims["userId"].(float64))
	user, err := store.GetUserByID(context.Background(), userId)
	if err != nil {
		return userClaims, fmt.Errorf("unknown userId")
	}

	// if we have a value in issued at, we compare against iat
	iat := int64(claims["iat"].(float64))
	if user.MinTokenIssuedTime.Valid {
		unitTs := user.MinTokenIssuedTime.Time.Unix()
		if unitTs > iat {
			// token was issued after the iat
			return userClaims, fmt.Errorf("token issued after iat=%d", iat)
		}
	}

	// generate claims
	var groups []int32
	if claims["groupIds"] != nil {
		for _, g := range claims["groupIds"].([]interface{}) {
			groups = append(groups, int32(g.(float64)))
		}
	}
	userClaims.GroupIds = groups

	var roles []RoleInfo
	if claims["roleInfo"] != nil {
		for _, role := range claims["roleInfo"].([]interface{}) {
			roleInfo := RoleInfo{RoleID: int32(role.(map[string]interface{})["role_id"].(float64)),
				IsAdmin:        role.(map[string]interface{})["is_admin"].(bool),
				GroupID:        int32(role.(map[string]interface{})["group_id"].(float64)),
				OrganizationID: int32(role.(map[string]interface{})["organization_id"].(float64))}

			roles = append(roles, roleInfo)
		}
	}
	userClaims.Roles = roles
	userClaims.UserId = int32(claims["userId"].(float64))

	userClaims.OrganizationId = int32(claims["orgId"].(float64))
	userClaims.NeedsPasswordChange = claims["needsPasswordChange"].(bool)

	return userClaims, nil
}

// VerifyRefreshToken verifies the refresh token string and returns the user ID
func VerifyRefreshToken(tokenString string, publicKey *rsa.PublicKey, store db.Store) (int32, error) {
	if publicKey == nil {
		return 0, fmt.Errorf("invalid key")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, status.Error(codes.InvalidArgument, "unexpected signing method")
		}
		return publicKey, nil
	})

	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	// validate that iat is on the past
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if !claims.VerifyIssuedAt(time.Now().Unix(), true) {
			return 0, fmt.Errorf("invalid token")
		}
	}

	// we have the user id, check if exists
	userId := int32(claims["userId"].(float64))
	_, err = store.GetUserByID(context.Background(), userId)
	if err != nil {
		return 0, fmt.Errorf("invalid token")
	}

	return userId, nil
}

// GetUserClaims returns the user claims for the given user
func GetUserClaims(ctx context.Context, store db.Store, userId int32) (UserClaims, error) {
	emptyClaims := UserClaims{}

	// read all information for user claims
	userInfo, err := store.GetUserByID(ctx, userId)
	if err != nil {
		return emptyClaims, fmt.Errorf("failed to read user")
	}

	// read groups and add id to claims
	gs, err := store.GetUserGroups(ctx, userId)
	if err != nil {
		return emptyClaims, fmt.Errorf("failed to get groups")
	}
	var groups []int32
	for _, g := range gs {
		groups = append(groups, g.ID)
	}

	// read roles and add details to claims
	rs, err := store.GetUserRoles(ctx, userId)
	if err != nil {
		return emptyClaims, fmt.Errorf("failed to get roles")
	}

	var roles []RoleInfo
	for _, r := range rs {
		roles = append(roles, RoleInfo{RoleID: r.ID, IsAdmin: r.IsAdmin, GroupID: r.GroupID.Int32, OrganizationID: r.OrganizationID})
	}

	claims := UserClaims{
		UserId:              userId,
		Roles:               roles,
		GroupIds:            groups,
		OrganizationId:      userInfo.OrganizationID,
		NeedsPasswordChange: userInfo.NeedsPasswordChange,
	}

	return claims, nil
}

var tokenContextKey struct{}

// GetClaimsFromContext returns the claims from the context, or an empty default
func GetClaimsFromContext(ctx context.Context) UserClaims {
	claims, ok := ctx.Value(tokenContextKey).(UserClaims)
	if !ok {
		return UserClaims{UserId: -1, OrganizationId: -1}
	}
	return claims
}

// WithClaimContext stores the specified UserClaim in the context.
func WithClaimContext(ctx context.Context, claims UserClaims) context.Context {
	return context.WithValue(ctx, tokenContextKey, claims)
}

// GetDefaultGroup returns the default group id for the user
func GetDefaultGroup(ctx context.Context) (int32, error) {
	claims := GetClaimsFromContext(ctx)
	if len(claims.GroupIds) != 1 {
		return 0, errors.New("cannot get default group")
	}
	return claims.GroupIds[0], nil
}

// IsAuthorizedForGroup returns true if the user is authorized for the given group
func IsAuthorizedForGroup(ctx context.Context, groupId int32) bool {
	claims := GetClaimsFromContext(ctx)

	return slices.Contains(claims.GroupIds, groupId)
}

// GetUserGroups returns all the groups where an user belongs to
func GetUserGroups(ctx context.Context) ([]int32, error) {
	claims := GetClaimsFromContext(ctx)
	return claims.GroupIds, nil
}
