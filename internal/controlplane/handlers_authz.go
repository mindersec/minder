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
	"strings"
	"time"

	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/util"
	mediator "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func (s *Server) parseTokenForAuthz(token string) (auth.UserClaims, error) {
	var claims auth.UserClaims
	// need to read pub key from file
	pubKeyData, err := s.cfg.Auth.GetAccessTokenPublicKey()
	if err != nil {
		return claims, fmt.Errorf("failed to read public key file")
	}

	userClaims, err := auth.VerifyToken(token, pubKeyData, s.store)
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

// AuthorizedOnOrg checks if the request is authorized for the given
// organization, and returns an error if the request is not authorized.
func AuthorizedOnOrg(ctx context.Context, orgId int32) error {
	claims, _ := ctx.Value(auth.TokenInfoKey).(auth.UserClaims)
	if isSuperadmin(claims) {
		return nil
	}
	opts := getRpcOptions(ctx)
	if opts.GetAuthScope() != mediator.ObjectOwner_OBJECT_OWNER_ORGANIZATION {
		zerolog.Ctx(ctx).Error().Msgf("Called IsOrgAuthorized on non-org method, should be %v", opts.GetAuthScope())
	}
	if claims.OrganizationId != orgId {
		return util.UserVisibleError(codes.PermissionDenied, "user is not authorized to access this organization")
	}
	isOwner := func(role auth.RoleInfo) bool {
		return role.GroupID == 0 && int32(role.OrganizationID) == orgId && role.IsAdmin
	}
	if opts.GetOwnerOnly() && !slices.ContainsFunc(claims.Roles, isOwner) {
		return util.UserVisibleError(codes.PermissionDenied, "user is not an administrator on this organization")
	}
	return nil
}

// AuthorizedOnGroup checks if the request is authorized for the given
// group, and returns an error if the request is not authorized.
func AuthorizedOnGroup(ctx context.Context, groupId int32) error {
	claims, _ := ctx.Value(auth.TokenInfoKey).(auth.UserClaims)
	if isSuperadmin(claims) {
		return nil
	}
	opts := getRpcOptions(ctx)
	if opts.GetAuthScope() != mediator.ObjectOwner_OBJECT_OWNER_GROUP {
		zerolog.Ctx(ctx).Error().Msgf("Called IsGroupAuthorized on non-group method, should be %v", opts.GetAuthScope())
	}

	if !slices.Contains(claims.GroupIds, groupId) {
		return util.UserVisibleError(codes.PermissionDenied, "user is not authorized to access this group")
	}
	isOwner := func(role auth.RoleInfo) bool {
		return int32(role.GroupID) == groupId && role.IsAdmin
	}
	// check if is admin of group
	if opts.GetOwnerOnly() && !slices.ContainsFunc(claims.Roles, isOwner) {
		return util.UserVisibleError(codes.PermissionDenied, "user is not an administrator on this group")
	}
	return nil
}

// AuthorizedOnUser checks if the request is authorized for the given
// user, and returns an error if the request is not authorized.
func AuthorizedOnUser(ctx context.Context, userId int32) error {
	claims, _ := ctx.Value(auth.TokenInfoKey).(auth.UserClaims)
	if isSuperadmin(claims) {
		return nil
	}
	opts := getRpcOptions(ctx)
	if opts.GetAuthScope() != mediator.ObjectOwner_OBJECT_OWNER_USER {
		zerolog.Ctx(ctx).Error().Msgf("Called IsUserAuthorized on non-user method, should be %v", opts.GetAuthScope())
	}

	if claims.UserId == userId {
		return nil
	}
	return util.UserVisibleError(codes.PermissionDenied, "user is not authorized to access this user")
}

// IsProviderCallAuthorized checks if the request is authorized
func (s *Server) IsProviderCallAuthorized(ctx context.Context, provider db.Provider, groupId int32) bool {
	if provider.GroupID != groupId {
		return false
	}

	// currently everything is github
	method, ok := grpc.Method(ctx)
	if !ok {
		return false
	}

	for _, item := range githubAuthorizations {
		if item == method {
			// check the github token
			encToken, _, err := s.GetProviderAccessToken(ctx, provider.Name, groupId, true)
			if err != nil {
				return false
			}

			// check if token is expired
			if encToken.Expiry.Unix() < time.Now().Unix() {
				// remove from the database and deny the request
				_ = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: provider.Name, GroupID: groupId})

				// remove from github
				err := auth.DeleteAccessToken(ctx, provider.Name, encToken.AccessToken)

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
	claims, err := server.parseTokenForAuthz(token)
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
