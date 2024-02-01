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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/util"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type rpcOptionsKey struct{}

func getRpcOptions(ctx context.Context) *minder.RpcOptions {
	// nil value default is okay here
	opts, _ := ctx.Value(rpcOptionsKey{}).(*minder.RpcOptions)
	return opts
}

// EntityContextProjectInterceptor is a server interceptor that sets up the entity context project
func EntityContextProjectInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
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

	server := info.Server.(*Server)

	ctx, err := populateEntityContext(ctx, server.store, server.authzClient, request)
	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

// ProjectAuthorizationInterceptor is a server interceptor that checks if a user is authorized on the requested project
func ProjectAuthorizationInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (any, error) {

	opts := getRpcOptions(ctx)

	if opts.GetTargetResource() != minder.TargetResource_TARGET_RESOURCE_PROJECT {
		if !opts.GetNoLog() {
			zerolog.Ctx(ctx).Info().Msgf("Bypassing project authorization")
		}
		return handler(ctx, req)
	}

	relation := opts.GetRelation()

	relationValue := relation.Descriptor().Values().ByNumber(relation.Number())
	if relationValue == nil {
		return nil, status.Errorf(codes.Internal, "error reading relation value %v", relation)
	}
	extension := proto.GetExtension(relationValue.Options(), minder.E_Name)
	relationName, ok := extension.(string)
	if !ok {
		return nil, status.Errorf(codes.Internal, "error getting name for requested relation %v", relation)
	}

	entityCtx := engine.EntityFromContext(ctx)
	server := info.Server.(*Server)

	if err := server.authzClient.Check(ctx, relationName, entityCtx.Project.ID); err != nil {
		return nil, util.UserVisibleError(codes.PermissionDenied, "user is not authorized to perform this operation")
	}

	return handler(ctx, req)
}

// populateEntityContext populates the project in the entity context, by looking at the proto context or
// fetching the default project
func populateEntityContext(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	in HasProtoContext,
) (context.Context, error) {
	if in.GetContext() == nil {
		return ctx, fmt.Errorf("context cannot be nil")
	}

	projectID, err := getProjectFromRequestOrDefault(ctx, store, authzClient, in)
	if err != nil {
		return ctx, err
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

func getProjectFromRequestOrDefault(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	in HasProtoContext,
) (uuid.UUID, error) {
	// Prefer the context message from the protobuf
	if in.GetContext().GetProject() != "" {
		requestedProject := in.GetContext().GetProject()
		parsedProjectID, err := uuid.Parse(requestedProject)
		if err != nil {
			return uuid.UUID{}, util.UserVisibleError(codes.InvalidArgument, "malformed project ID")
		}
		return parsedProjectID, nil
	}

	subject := auth.GetUserSubjectFromContext(ctx)

	userInfo, err := store.GetUserBySubject(ctx, subject)
	if err != nil {
		// Note that we're revealing that the user is not registered in minder
		// since the caller has a valid token (this is checked in earlier middleware).
		// Therefore, we assume it's safe output that the user is not found.
		return uuid.UUID{}, util.UserVisibleError(codes.NotFound, "user not found")
	}
	projects, err := authzClient.ProjectsForUser(ctx, userInfo.IdentitySubject)
	if err != nil {
		return uuid.UUID{}, status.Errorf(codes.Internal, "cannot find projects for user")
	}

	if len(projects) == 0 {
		return uuid.UUID{}, util.UserVisibleError(codes.PermissionDenied, "User has no role grants in projects")
	}

	if len(projects) != 1 {
		return uuid.UUID{}, util.UserVisibleError(codes.PermissionDenied, "Cannot determine default project. Please specify one.")
	}

	return projects[0], nil
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
func (s *Server) ListRoleAssignments(
	ctx context.Context,
	_ *minder.ListRoleAssignmentsRequest,
) (*minder.ListRoleAssignmentsResponse, error) {
	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	as, err := s.authzClient.AssignmentsToProject(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	return &minder.ListRoleAssignmentsResponse{
		RoleAssignments: as,
	}, nil
}

// AssignRole assigns a role to a user on a project.
// Note that this assumes that the request has already been authorized.
func (s *Server) AssignRole(ctx context.Context, req *minder.AssignRoleRequest) (*minder.AssignRoleResponse, error) {
	// Request Validation
	role := req.GetRoleAssignment().GetRole()
	sub := req.GetRoleAssignment().GetSubject()

	if role == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "role and subject must be specified")
	}

	if sub == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "subject must be specified")
	}

	// Parse role (this also validates)
	authzrole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	respProj := projectID.String()

	if sub != "" {
		// Verify if user exists
		if _, err := s.store.GetUserBySubject(ctx, sub); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "User not found")
			}
			return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
		}

		if err := s.authzClient.Write(ctx, sub, authzrole, projectID); err != nil {
			return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
		}

		return &minder.AssignRoleResponse{
			RoleAssignment: &minder.RoleAssignment{
				Role:    role,
				Subject: sub,
				Project: &respProj,
			},
		}, nil
	}

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error starting transaction: %v", err)
	}
	defer tx.Rollback()

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}

	return &minder.AssignRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role,
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

// CreateRoleMapping creates a role mapping for the given project
func (s *Server) CreateRoleMapping(
	ctx context.Context,
	req *minder.CreateRoleMappingRequest,
) (*minder.CreateRoleMappingResponse, error) {
	// Request Validation
	role := req.GetRoleMapping().GetRole()
	claims := req.GetRoleMapping().GetClaimsToMatch()

	if role == "" || claims == nil || len(claims.AsMap()) == 0 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "role and claims must be specified")
	}

	// Parse role (this also validates)
	authzrole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	jsonclaims, err := json.Marshal(claims)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error marshalling claims: %v", err)
	}

	mrg, err := s.store.AddMappedRoleGrant(ctx, db.AddMappedRoleGrantParams{
		ProjectID:     projectID,
		Role:          authzrole.String(),
		ClaimMappings: jsonclaims,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating role mapping: %v", err)
	}

	mrgID := mrg.ID.String()
	return &minder.CreateRoleMappingResponse{
		RoleMapping: &minder.RoleMapping{
			Id:            &mrgID,
			Role:          role,
			ClaimsToMatch: claims,
		},
	}, nil
}

// ListRoleMappings returns the list of role mappings for the given project
func (s *Server) ListRoleMappings(
	ctx context.Context,
	_ *minder.ListRoleMappingsRequest,
) (*minder.ListRoleMappingsResponse, error) {
	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	mrgs, err := s.store.ListMappedRoleGrants(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error listing role mappings: %v", err)
	}

	resp := minder.ListRoleMappingsResponse{
		RoleMappings: make([]*minder.RoleMapping, 0, len(mrgs)),
	}
	for _, mrg := range mrgs {
		mrgID := mrg.ID.String()
		mappings := &structpb.Struct{}
		prj := projectID.String()

		if err := json.Unmarshal(mrg.ClaimMappings, mappings); err != nil {
			return nil, status.Errorf(codes.Internal, "error unmarshalling role mapping: %v", err)
		}

		rm := &minder.RoleMapping{
			Id:            &mrgID,
			Project:       &prj,
			Role:          mrg.Role,
			ClaimsToMatch: mappings,
		}

		if mrg.ResolvedSubject.Valid {
			sub := mrg.ResolvedSubject.String
			rm.ResolvedSubject = &sub
		}

		resp.RoleMappings = append(resp.RoleMappings, rm)
	}

	return &resp, nil
}

// DeleteRoleMapping deletes a role mapping from the given project
func (s *Server) DeleteRoleMapping(
	ctx context.Context,
	req *minder.DeleteRoleMappingRequest,
) (*minder.DeleteRoleMappingResponse, error) {
	// Request Validation
	mappingID := req.GetId()

	if mappingID == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "role mapping ID must be specified")
	}

	mrgUUID, err := uuid.Parse(mappingID)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "malformed role mapping ID")
	}

	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	_, err = s.store.DeleteMappedRoleGrant(ctx, db.DeleteMappedRoleGrantParams{
		ID:        mrgUUID,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error deleting role mapping: %v", err)
	}

	return &minder.DeleteRoleMappingResponse{
		Id: mappingID,
	}, nil
}
