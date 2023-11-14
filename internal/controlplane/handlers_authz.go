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

	"github.com/google/uuid"
	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type rpcOptionsKey struct{}

func withRpcOptions(ctx context.Context, opts *minder.RpcOptions) context.Context {
	return context.WithValue(ctx, rpcOptionsKey{}, opts)
}

func getRpcOptions(ctx context.Context) *minder.RpcOptions {
	// nil value default is okay here
	opts, _ := ctx.Value(rpcOptionsKey{}).(*minder.RpcOptions)
	return opts
}

// checks if an user is superadmin
func isSuperadmin(claims auth.UserPermissions) bool {
	return claims.IsStaff
}

func containsSuperadminRole(openIdToken openid.Token) bool {
	if realmAccess, ok := openIdToken.Get("realm_access"); ok {
		if realms, ok := realmAccess.(map[string]interface{}); ok {
			if roles, ok := realms["roles"]; ok {
				if userRoles, ok := roles.([]interface{}); ok {
					if slices.Contains(userRoles, "superadmin") {
						return true
					}
				}
			}
		}
	}
	return false
}

// lookupUserPermissions returns the user permissions from the database for the given user
func lookupUserPermissions(ctx context.Context, store db.Store, tok openid.Token) (auth.UserPermissions, error) {
	emptyPermissions := auth.UserPermissions{}

	// read all information for user claims
	userInfo, err := store.GetUserBySubject(ctx, tok.Subject())
	if err != nil {
		return emptyPermissions, fmt.Errorf("failed to read user")
	}

	// read groups and add id to claims
	gs, err := store.GetUserProjects(ctx, userInfo.ID)
	if err != nil {
		return emptyPermissions, fmt.Errorf("failed to get groups")
	}
	var groups []uuid.UUID
	for _, g := range gs {
		groups = append(groups, g.ID)
	}

	// read roles and add details to claims
	rs, err := store.GetUserRoles(ctx, userInfo.ID)
	if err != nil {
		return emptyPermissions, fmt.Errorf("failed to get roles")
	}

	var roles []auth.RoleInfo
	for _, r := range rs {
		rif := auth.RoleInfo{
			RoleID:         r.ID,
			IsAdmin:        r.IsAdmin,
			OrganizationID: r.OrganizationID,
		}
		if r.ProjectID.Valid {
			pID := r.ProjectID.UUID
			rif.ProjectID = &pID
		}
		roles = append(roles, rif)
	}

	claims := auth.UserPermissions{
		UserId:         userInfo.ID,
		Roles:          roles,
		ProjectIds:     groups,
		OrganizationId: userInfo.OrganizationID,
		IsStaff:        containsSuperadminRole(tok),
	}

	return claims, nil
}

// AuthorizedOnProject checks if the request is authorized for the given
// group, and returns an error if the request is not authorized.
func AuthorizedOnProject(ctx context.Context, projectID uuid.UUID) error {
	claims := auth.GetPermissionsFromContext(ctx)
	if isSuperadmin(claims) {
		return nil
	}
	opts := getRpcOptions(ctx)
	if opts.GetAuthScope() != minder.ObjectOwner_OBJECT_OWNER_PROJECT {
		return status.Errorf(codes.Internal, "Called IsProjectAuthorized on non-group method, should be %v", opts.GetAuthScope())
	}

	if !slices.Contains(claims.ProjectIds, projectID) {
		return util.UserVisibleError(codes.PermissionDenied, "user is not authorized to access this project")
	}
	isOwner := func(role auth.RoleInfo) bool {
		if role.ProjectID == nil {
			return false
		}
		return *role.ProjectID == projectID && role.IsAdmin
	}
	// check if is admin of group
	if opts.GetOwnerOnly() && !slices.ContainsFunc(claims.Roles, isOwner) {
		return util.UserVisibleError(codes.PermissionDenied, "user is not an administrator on this project")
	}
	return nil
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

	parsedToken, err := server.vldtr.ParseAndValidate(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	// get user authorities from the database
	// ignore any error because the user may not exist yet
	authorities, _ := lookupUserPermissions(ctx, server.store, parsedToken)

	if opts.GetRootAdminOnly() && !isSuperadmin(authorities) {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized")
	}

	ctx = auth.WithPermissionsContext(ctx, authorities)
	return handler(ctx, req)
}

func optionsForMethod(info *grpc.UnaryServerInfo) (*minder.RpcOptions, error) {
	formattedName := strings.ReplaceAll(info.FullMethod[1:], "/", ".")
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(formattedName))
	if err != nil {
		return nil, fmt.Errorf("unable to find descriptor for %q: %w", formattedName, err)
	}
	extension := proto.GetExtension(descriptor.Options(), minder.E_RpcOptions)
	opts, ok := extension.(*minder.RpcOptions)
	if !ok {
		return nil, fmt.Errorf("couldn't decode option for %q, wrong type: %T", formattedName, extension)
	}
	return opts, nil
}
