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
	"strings"
	"time"

	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/auth"
	mediator "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	github "github.com/stacklok/mediator/pkg/providers/github"
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

type rpcOptionsKey struct{}

func withRpcOptions(ctx context.Context, opts *mediator.RpcOptions) context.Context {
	return context.WithValue(ctx, rpcOptionsKey{}, opts)
}

func getRpcOptions(ctx context.Context) *mediator.RpcOptions {
	// nil value default is okay here
	opts, _ := ctx.Value(rpcOptionsKey{}).(*mediator.RpcOptions)
	return opts
}

var githubAuthorizations = []string{
	"/mediator.v1.RepositoryService/AddRepository",
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

// IsRequestAuthorized checks if the request is authorized
// nolint:gocyclo
func IsRequestAuthorized(ctx context.Context, value int32) bool {
	claims, _ := ctx.Value(auth.TokenInfoKey).(auth.UserClaims)
	if isSuperadmin(claims) {
		return true
	}
	opts := getRpcOptions(ctx)

	switch opts.GetAuthScope() {
	case mediator.ObjectOwner_OBJECT_OWNER_ORGANIZATION:
		if claims.OrganizationId != value {
			return false
		}
		if opts.GetOwnerOnly() && !isUserAdmin(claims, "OrganizationId", value) {
			return false
		}
		return true
	case mediator.ObjectOwner_OBJECT_OWNER_GROUP:
		if !slices.Contains(claims.GroupIds, value) {
			return false
		}

		// check if is admin of group
		if opts.GetOwnerOnly() && !isUserAdmin(claims, "GroupId", value) {
			return false
		}
		return true
	case mediator.ObjectOwner_OBJECT_OWNER_USER:
		if claims.UserId == 0 {
			return false
		}
		return true
	case mediator.ObjectOwner_OBJECT_OWNER_UNSPECIFIED:
		fallthrough
	default:
		return false
	}
}

// IsProviderCallAuthorized checks if the request is authorized
func (s *Server) IsProviderCallAuthorized(ctx context.Context, provider string, groupId int32) bool {
	// currently everything is github
	method, ok := grpc.Method(ctx)
	if !ok {
		return false
	}

	for _, item := range githubAuthorizations {
		if item == method {
			// check the github token
			encToken, _, err := s.GetProviderAccessToken(ctx, provider, groupId, true)
			if err != nil {
				return false
			}

			// check if token is expired
			if encToken.Expiry.Unix() < time.Now().Unix() {
				// remove from the database and deny the request
				_ = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: github.Github, GroupID: groupId})

				// remove from github
				err := auth.DeleteAccessToken(ctx, provider, encToken.AccessToken)

				if err != nil {
					zerolog.Ctx(ctx).Error().Msgf("Error deleting access token: %v", err)
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

	opts, err := optionsForMethod(info)
	if err != nil {
		// Fail closed safely, rather than log and proceed.
		return nil, status.Errorf(codes.Internal, "Error getting options for method: %v", err)
	}

	ctx = withRpcOptions(ctx, opts)

	if opts.GetAnonymous() {
		if !opts.GetNoLog() {
			zerolog.Ctx(ctx).Info().Msgf("Bypassing authentication")
		}
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
	if claims.NeedsPasswordChange && info.FullMethod != "/mediator.v1.UserService/UpdatePassword" {
		return nil, util.UserVisibleError(codes.Unauthenticated, "password change required")
	}

	if opts.GetRootAdminOnly() && !isSuperadmin(claims) {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized")
	}

	ctx = context.WithValue(ctx, auth.TokenInfoKey, claims)
	return handler(ctx, req)
}

func optionsForMethod(info *grpc.UnaryServerInfo) (*mediator.RpcOptions, error) {
	formattedName := strings.ReplaceAll(info.FullMethod[1:], "/", ".")
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(formattedName))
	if err != nil {
		return nil, fmt.Errorf("Unable to find descriptor for %q: %w", formattedName, err)
	}
	extension := proto.GetExtension(descriptor.Options(), mediator.E_RpcOptions)
	opts, ok := extension.(*mediator.RpcOptions)
	if !ok {
		return nil, fmt.Errorf("Couldn't decode option for %q, wrong type: %T", formattedName, extension)
	}
	return opts, nil
}
