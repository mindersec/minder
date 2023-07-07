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
	"time"

	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	"/mediator.v1.AuthService/LogIn",
	"/mediator.v1.HealthService/CheckHealth",
	"/mediator.v1.OAuthService/ExchangeCodeForTokenCLI",
	"/mediator.v1.OAuthService/ExchangeCodeForTokenWEB",
}

var superAdminMethods = []string{
	"/mediator.v1.OrganizationService/CreateOrganization",
	"/mediator.v1.OrganizationService/GetOrganizations",
	"/mediator.v1.OrganizationService/DeleteOrganization",
	"/mediator.v1.AuthService/RevokeTokens",
	"/mediator.v1.AuthService/RevokeUserToken",
	"/mediator.v1.OAuthService/RevokeOauthTokens",
	"/mediator.v1.UserService/GetUsers",
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
		"/mediator.v1.GroupService/GetGroupById": {
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
		"/mediator.v1.RoleService/CreateRoleByOrganization": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/CreateRoleByGroup": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/DeleteRole": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/GetRoles": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/GetRolesByGroup": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/GetRoleById": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RoleService/GetRoleByName": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/CreateUser": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/DeleteUser": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUsersByOrganization": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUsersByGroup": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUserById": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUserByUserName": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.UserService/GetUser": {
			"claimField": "UserId",
			"isAdmin":    false,
		},
	},
	{
		"/mediator.v1.UserService/UpdateUser": {
			"claimField": "UserId",
			"isAdmin":    false,
		},
	},
	{
		"/mediator.v1.UserService/GetUserByEmail": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.OAuthService/GetAuthorizationURL": {
			"claimField": "OrganizationId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.OAuthService/RevokeOauthGroupToken": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RepositoryService/ListRepositories": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.RepositoryService/GetRepository": {
			"claimField": "GroupId",
			"isAdmin":    false,
		},
	},
	{
		"/mediator.v1.RepositoryService/RegisterRepository": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.PolicyService/CreatePolicy": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.PolicyService/DeletePolicy": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.PolicyService/GetPolicies": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
	{
		"/mediator.v1.PolicyService/GetPolicy": {
			"claimField": "GroupId",
			"isAdmin":    true,
		},
	},
}

var githubAuthorizations = []string{
	"/mediator.v1.RepositoryService/AddRepository",
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

// checks if an user is superadmin
func isSuperadmin(claims auth.UserClaims) bool {
	// need to check that has a role that belongs to org 1 generally and is admin
	for _, role := range claims.Roles {
		if role.OrganizationID == 1 && role.GroupID == 0 && role.IsAdmin {
			return true
		}
	}
	return false
}

func isUserAdmin(claims auth.UserClaims, claimsField string, claimsValue int32) bool {
	if claimsField == "OrganizationId" {
		// need to check for a role that is only for org and has admin
		for _, role := range claims.Roles {
			if role.GroupID == 0 && int32(role.OrganizationID) == claimsValue && role.IsAdmin {
				return true
			}
		}
	} else if claimsField == "GroupId" {
		// need to check for a role that is only for group and has admin
		for _, role := range claims.Roles {
			if int32(role.GroupID) == claimsValue && role.IsAdmin {
				return true
			}
		}
	}
	return false
}

func isMethodAuthorized(ctx context.Context, claims auth.UserClaims) bool {
	// superadmin is authorized to everything
	if isSuperadmin(claims) {
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
// nolint:gocyclo
func IsRequestAuthorized(ctx context.Context, value int32) bool {
	claims, _ := ctx.Value(auth.TokenInfoKey).(auth.UserClaims)
	if isSuperadmin(claims) {
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

				if claimField == "OrganizationId" {
					// check if user belongs to same org
					if claims.OrganizationId != value {
						return false
					}
					// check if is admin of org
					if isAdmin && !isUserAdmin(claims, "OrganizationId", value) {
						return false
					}

				} else if claimField == "GroupId" {
					// check if user is in the list of groups
					found := false
					for _, group := range claims.GroupIds {
						if group == value {
							found = true
							break
						}
					}
					if !found {
						return false
					}

					// check if is admin of group
					if isAdmin && !isUserAdmin(claims, "GroupId", value) {
						return false
					}
				} else if claimField == "UserId" {
					// check that user id is not 0
					if claims.UserId == 0 {
						return false
					}
				} else {
					// no claim field to match
					return false
				}

				return true
			}
		}
	}
	return true
}

// IsProviderCallAuthorized checks if the request is authorized
func IsProviderCallAuthorized(ctx context.Context, store db.Store, provider string, groupId int32) bool {
	// currently everything is github
	method, ok := grpc.Method(ctx)
	if !ok {
		return false
	}

	for _, item := range githubAuthorizations {
		if item == method {
			// check the github token
			encToken, err := GetProviderAccessToken(ctx, store, provider, groupId)
			if err != nil {
				return false
			}

			// check if token is expired
			if encToken.Expiry.Unix() < time.Now().Unix() {
				// remove from the database and deny the request
				_ = store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: auth.Github, GroupID: groupId})

				// remove from github
				err := auth.DeleteAccessToken(ctx, provider, encToken.AccessToken)

				if err != nil {
					log.Error().Msgf("Error deleting access token: %v", err)
				}
				return false
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

	// check if we need a password change
	method, ok := grpc.Method(ctx)
	if !ok {
		// no method called and did not bypass auth, return false
		return nil, status.Errorf(codes.Unauthenticated, "no method called")
	}

	if claims.NeedsPasswordChange && method != "/mediator.v1.UserService/UpdatePassword" {
		return nil, status.Errorf(codes.Unauthenticated, "password change required")
	}

	if isSuperadmin(claims) {
		// is authorized to everything
		ctx = context.WithValue(ctx, auth.TokenInfoKey, claims)
	} else {
		// Check if the current method needs to have a superadmin role
		isAuthorized := isMethodAuthorized(ctx, claims)
		if !isAuthorized {
			return nil, status.Errorf(codes.PermissionDenied, "user not authorized")
		}
	}

	ctx = context.WithValue(ctx, auth.TokenInfoKey, claims)
	return handler(ctx, req)
}
