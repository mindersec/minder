// Copyright 2023 Stacklok, Inc
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

package controlplane

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TokenInfoKey is the key used to store the token info in the context
var TokenInfoKey struct{}

func parseToken(token string, store db.Store) (auth.UserClaims, error) {
	var claims auth.UserClaims
	// need to read pub key from file
	publicKeyPath := viper.GetString("auth.access_token_public_key")
	if publicKeyPath == "" {
		return claims, fmt.Errorf("could not read public key")
	}
	pubKeyData, err := os.ReadFile(filepath.Clean(publicKeyPath))
	if err != nil {
		return claims, fmt.Errorf("failed to read public key file")
	}

	userClaims, err := auth.VerifyToken(token, pubKeyData, store)
	if err != nil {
		return claims, fmt.Errorf("failed to verify token: %v", err)
	}
	return userClaims, nil
}

// List of methods that bypass authentication
var authBypassMethods = []string{
	"/mediator.v1.LogInService/LogIn",
	"/mediator.v1.HealthService/CheckHealth",
	"/mediator.v1.OAuthService/ExchangeCodeForTokenCLI",
	"/mediator.v1.OAuthService/ExchangeCodeForTokenWEB",
}

var superAdminMethods = []string{
	"/mediator.v1.OrganizationService/CreateOrganization",
	"/mediator.v1.OrganizationService/GetOrganizations",
	"/mediator.v1.OrganizationService/DeleteOrganization",
	"/mediator.v1.RevokeTokensService/RevokeTokens",
	"/mediator.v1.RevokeTokensService/RevokeUserToken",
}

var resourceAuthorizations = []map[string]map[string]interface{}{
	{
		"/mediator.v1.OrganizationService/GetOrganization": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.OrganizationService/GetOrganizationByName": {
			"claimField": "OrganizationId",
			"isAdmin":    false,
		},
	},
	{
		"/mediator.v1.GroupService/CreateGroup": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.GroupService/GetGroups": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.GroupService/GetGroupByName": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.OrganizationService/GetGroupById": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.GroupService/DeleteGroup": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/CreateRole": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/DeleteRole": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/GetRoles": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/GetRoleById": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/GetRoleByName": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/CreateUser": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/DeleteUser": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUsers": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUserById": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUserByUserName": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUserByEmail": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
}

func canBypassAuth(ctx context.Context) bool {
	// Extract the gRPC method name from the context
	method, ok := grpc.Method(ctx)
	if !ok {
		// no method called, can bypass auth
		return true
	}

	// Check if the current method is in the list of bypass methods
	for _, bypassMethod := range authBypassMethods {
		if bypassMethod == method {
			return true
		}
	}
	return false
}

func isMethodAuthorized(ctx context.Context, claims auth.UserClaims) bool {
	// superadmin is authorized to everything
	if claims.IsSuperadmin {
		return true
	}
	// Extract the gRPC method name from the context
	method, ok := grpc.Method(ctx)
	if !ok {
		// no method called and did not bypass auth, return false
		return false
	}

	// check if method is on superadmin ones, and fail
	for _, bypassMethod := range superAdminMethods {
		if bypassMethod == method {
			return false
		}
	}

	return true

}

// IsRequestAuthorized checks if the request is authorized
func IsRequestAuthorized(ctx context.Context, value int32) bool {
	claims, _ := ctx.Value(TokenInfoKey).(auth.UserClaims)
	if claims.IsSuperadmin {
		return true
	}
	method, ok := grpc.Method(ctx)
	if !ok {
		return false
	}

	// grant permissions depending on request type and claims
	for _, authorization := range resourceAuthorizations {
		for path, data := range authorization {
			// method matches, now we need to check if the request has the field
			if path == method {
				// now check if claims match
				claimField := data["claimField"].(string)
				isAdmin := data["isAdmin"].(bool)

				claimsObj := reflect.ValueOf(claims)
				claimsValue := claimsObj.FieldByName(claimField).Interface().(int32)

				// if resources do not match, do not authorize
				if claimsValue != value {
					return false
				}

				// if needs admin role but is not admin, do not authorize
				if isAdmin && !claims.IsAdmin {
					return false
				}
				return true
			}
		}
	}
	return true
}

// AuthUnaryInterceptor is a server interceptor for authentication
func AuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (any, error) {
	// bypass auth
	canBypass := canBypassAuth(ctx)
	if canBypass {
		// If the method is in the bypass list, return the context as is without authentication
		log.Info().Msgf("Bypassing authentication")
		return handler(ctx, req)
	}

	token, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "no auth token: %v", err)
	}

	server := info.Server.(*Server)
	claims, err := parseToken(token, server.store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}
	if claims.IsSuperadmin {
		// is authorized to everything
		ctx := context.WithValue(ctx, TokenInfoKey, claims)
		return handler(ctx, req)
	}

	// Check if the current method needs to have a superadmin role
	isAuthorized := isMethodAuthorized(ctx, claims)
	if !isAuthorized {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized")
	}

	ctx = context.WithValue(ctx, TokenInfoKey, claims)
	return handler(ctx, req)
}
