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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/util"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type rpcOptionsKey struct{}

func getRpcOptions(ctx context.Context) *minder.RpcOptions {
	// nil value default is okay here
	opts, _ := ctx.Value(rpcOptionsKey{}).(*minder.RpcOptions)
	return opts
}

// lookupUserPermissions returns the user permissions from the database for the given user
func lookupUserPermissions(ctx context.Context, store db.Store) auth.UserPermissions {
	emptyPermissions := auth.UserPermissions{}

	subject := auth.GetUserSubjectFromContext(ctx)

	// Attach the login sha for telemetry usage (hash of the user subject from the JWT)
	loginSHA := sha256.Sum256([]byte(subject))
	logger.BusinessRecord(ctx).LoginHash = hex.EncodeToString(loginSHA[:])

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

// authorizedOnProject checks if the request is authorized for the given
// project, and returns an error if the request is not authorized.
func authorizedOnProject(ctx context.Context, projectID uuid.UUID) error {
	claims := auth.GetPermissionsFromContext(ctx)
	opts := getRpcOptions(ctx)
	if opts.GetTargetResource() != minder.TargetResource_TARGET_RESOURCE_PROJECT {
		return status.Errorf(codes.Internal, "Called IsProjectAuthorized on non-project method, should be %v", opts.GetTargetResource())
	}

	// call openFGA using the relation and project ID
	// opts.GetRelation()

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

// PermissionsContextUnaryInterceptor is a server interceptor that sets up the user permissions
func PermissionsContextUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (any, error) {

	server := info.Server.(*Server)

	// get user authorities from the database
	// ignore any error because the user may not exist yet
	authorities := lookupUserPermissions(ctx, server.store)

	ctx = auth.WithPermissionsContext(ctx, authorities)
	return handler(ctx, req)
}

// EntityContextProjectInterceptor is a server interceptor that sets up the entity context project
func EntityContextProjectInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (any, error) {

	opts := getRpcOptions(ctx)

	if opts.GetTargetResource() == minder.TargetResource_TARGET_RESOURCE_UNSPECIFIED {
		return nil, status.Error(codes.Internal, "cannot perform authorization, because target resource is unspecified")
	}

	if opts.GetTargetResource() != minder.TargetResource_TARGET_RESOURCE_PROJECT {
		if !opts.GetNoLog() {
			zerolog.Ctx(ctx).Info().Msgf("Bypassing setting up context")
		}
		return handler(ctx, req)
	}

	request, ok := req.(HasProtoContext)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Error extracting context from request")
	}

	ctx, err := populateEntityContext(ctx, request)
	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

// ProjectAuthorizationInterceptor is a server interceptor that checks if a user is authorized on the requested project
func ProjectAuthorizationInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (any, error) {

	opts := getRpcOptions(ctx)

	if opts.GetTargetResource() != minder.TargetResource_TARGET_RESOURCE_PROJECT {
		if !opts.GetNoLog() {
			zerolog.Ctx(ctx).Info().Msgf("Bypassing project authorization")
		}
		return handler(ctx, req)
	}

	entityCtx := engine.EntityFromContext(ctx)
	if err := authorizedOnProject(ctx, entityCtx.Project.ID); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

// populateEntityContext populates the project in the entity context, by looking at the proto context or
// fetching the default project
func populateEntityContext(ctx context.Context, in HasProtoContext) (context.Context, error) {
	if in.GetContext() == nil {
		return ctx, fmt.Errorf("context cannot be nil")
	}

	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return ctx, fmt.Errorf("cannot get project from context: %v", err)
	}

	// don't look up default provider until user has been authorized
	providerName := in.GetContext().GetProvider()

	entityCtx := &engine.EntityContext{
		Project: engine.Project{
			ID: projectID,
		},
		Provider: engine.Provider{
			Name: providerName,
		},
	}

	return engine.WithEntityContext(ctx, entityCtx), nil
}

func getProjectFromRequestOrDefault(ctx context.Context, in HasProtoContext) (uuid.UUID, error) {
	// Prefer the context message from the protobuf
	if in.GetContext().GetProject() != "" {
		requestedProject := in.GetContext().GetProject()
		parsedProjectID, err := uuid.Parse(requestedProject)
		if err != nil {
			return uuid.UUID{}, util.UserVisibleError(codes.InvalidArgument, "malformed project ID")
		}
		return parsedProjectID, nil
	}

	permissions := auth.GetPermissionsFromContext(ctx)
	if len(permissions.ProjectIds) != 1 {
		return uuid.UUID{}, status.Errorf(codes.InvalidArgument, "cannot get default project")
	}
	return permissions.ProjectIds[0], nil
}

// Permissions API
// ensure interface implementation
var _ minder.PermissionsServiceServer = (*Server)(nil)

// ListRoles returns the list of available roles for the minder instance
func (*Server) ListRoles(_ context.Context, _ *minder.ListRolesRequest) (*minder.ListRolesResponse, error) {
	resp := minder.ListRolesResponse{
		Roles: make([]*minder.Role, 0, len(authz.AllRoles)),
	}
	for role, desc := range authz.AllRoles {
		resp.Roles = append(resp.Roles, &minder.Role{
			Name:        role.String(),
			Description: desc,
		})
	}

	return &resp, nil
}

// ListRoleAssignments returns the list of role assignments for the given project
func (*Server) ListRoleAssignments(
	context.Context,
	*minder.ListRoleAssignmentsRequest,
) (*minder.ListRoleAssignmentsResponse, error) {
	return nil, nil
}

// AssignRole assigns a role to a user on a project.
// Note that this assumes that the request has already been authorized.
func (s *Server) AssignRole(ctx context.Context, req *minder.AssignRoleRequest) (*minder.AssignRoleResponse, error) {
	// Request Validation
	role := req.GetRoleAssignment().GetRole()
	sub := req.GetRoleAssignment().GetSubject()

	if role == "" || sub == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "role and subject must be specified")
	}

	// Parse role (this also validates)
	authzrole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// Verify if user exists
	if _, err := s.store.GetUserBySubject(ctx, sub); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "User not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
	}

	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	if err := s.authzClient.Write(ctx, sub, authzrole, projectID); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := projectID.String()
	return &minder.AssignRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role,
			Subject: sub,
			Project: &respProj,
		},
	}, nil
}

// RemoveRole removes a role from a user on a project
// Note that this assumes that the request has already been authorized.
func (s *Server) RemoveRole(ctx context.Context, req *minder.RemoveRoleRequest) (*minder.RemoveRoleResponse, error) {
	// Request Validation
	role := req.GetRoleAssignment().GetRole()
	sub := req.GetRoleAssignment().GetSubject()

	if role == "" || sub == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "role and subject must be specified")
	}

	// Parse role (this also validates)
	authzrole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// Verify if user exists
	if _, err := s.store.GetUserBySubject(ctx, sub); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "User not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
	}

	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	if err := s.authzClient.Delete(ctx, sub, authzrole, projectID); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := projectID.String()
	return &minder.RemoveRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role,
			Subject: sub,
			Project: &respProj,
		},
	}, nil
}
