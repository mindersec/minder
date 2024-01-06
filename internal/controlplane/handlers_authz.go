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

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type rpcOptionsKey struct{}

func getRpcOptions(ctx context.Context) *minder.RpcOptions {
	// nil value default is okay here
	opts, _ := ctx.Value(rpcOptionsKey{}).(*minder.RpcOptions)
	return opts
}

// checks if an user is superadmin
func isSuperadmin(claims auth.UserPermissions) bool {
	return claims.IsStaff
}

// lookupUserPermissions returns the user permissions from the database for the given user
func lookupUserPermissions(ctx context.Context, store db.Store) auth.UserPermissions {
	emptyPermissions := auth.UserPermissions{}

	subject := auth.GetUserSubjectFromContext(ctx)

	// read all information for user claims
	userInfo, err := store.GetUserBySubject(ctx, subject)
	if err != nil {
		return emptyPermissions
	}

	// read projects and add id to claims
	gs, err := store.GetUserProjects(ctx, userInfo.ID)
	if err != nil {
		return emptyPermissions
	}
	var projects []uuid.UUID
	for _, g := range gs {
		projects = append(projects, g.ID)
	}

	// read roles and add details to claims
	rs, err := store.GetUserRoles(ctx, userInfo.ID)
	if err != nil {
		return emptyPermissions
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
		ProjectIds:     projects,
		OrganizationId: userInfo.OrganizationID,
	}

	return claims
}

// AuthorizedOnProject checks if the request is authorized for the given
// project, and returns an error if the request is not authorized.
func AuthorizedOnProject(ctx context.Context, projectID uuid.UUID) error {
	claims := auth.GetPermissionsFromContext(ctx)
	if isSuperadmin(claims) {
		return nil
	}
	opts := getRpcOptions(ctx)
	if opts.GetAuthScope() != minder.ObjectOwner_OBJECT_OWNER_PROJECT {
		return status.Errorf(codes.Internal, "Called IsProjectAuthorized on non-project method, should be %v", opts.GetAuthScope())
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
	// check if is admin of project
	if opts.GetOwnerOnly() && !slices.ContainsFunc(claims.Roles, isOwner) {
		return util.UserVisibleError(codes.PermissionDenied, "user is not an administrator on this project")
	}
	return nil
}

// AuthorizationUnaryInterceptor is a server interceptor that sets up the user permissions
func AuthorizationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (any, error) {

	server := info.Server.(*Server)

	// get user authorities from the database
	// ignore any error because the user may not exist yet
	authorities := lookupUserPermissions(ctx, server.store)

	ctx = auth.WithPermissionsContext(ctx, authorities)
	return handler(ctx, req)
}
