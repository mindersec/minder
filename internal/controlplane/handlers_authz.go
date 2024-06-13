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
	"encoding/base64"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/flags"
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

	server, ok := info.Server.(*Server)
	if !ok {
		return nil, status.Errorf(codes.Internal, "error casting serrver for request handling")
	}

	ctx, err := populateEntityContext(ctx, server.store, server.authzClient, req)
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
		zerolog.Ctx(ctx).Error().Err(err).Msg("authorization check failed")
		return nil, util.UserVisibleError(
			codes.PermissionDenied, "user %q is not authorized to perform this operation on project %q",
			auth.GetUserSubjectFromContext(ctx), entityCtx.Project.ID)
	}

	return handler(ctx, req)
}

// populateEntityContext populates the project in the entity context, by looking at the proto context or
// fetching the default project
func populateEntityContext(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	req any,
) (context.Context, error) {
	projectID, err := getProjectIDFromContext(req)
	if err != nil {
		if errors.Is(err, ErrNoProjectInContext) {
			projectID, err = getDefaultProjectID(ctx, store, authzClient)
			if err != nil {
				return ctx, err
			}
		} else {
			return ctx, err
		}
	}

	entityCtx := &engine.EntityContext{
		Project: engine.Project{
			ID: projectID,
		},
		Provider: engine.Provider{
			Name: getProviderFromContext(req),
		},
	}

	return engine.WithEntityContext(ctx, entityCtx), nil
}

func getProjectIDFromContext(req any) (uuid.UUID, error) {
	switch req := req.(type) {
	case HasProtoContextV2Compat:
		return getProjectFromContextV2Compat(req)
	case HasProtoContextV2:
		return getProjectFromContextV2(req)
	case HasProtoContext:
		return getProjectFromContext(req)
	default:
		return uuid.Nil, status.Errorf(codes.Internal, "Error extracting context from request")
	}
}

func getProviderFromContext(req any) string {
	switch req := req.(type) {
	case HasProtoContextV2Compat:
		if req.GetContextV2().GetProvider() != "" {
			return req.GetContextV2().GetProvider()
		}
		return req.GetContext().GetProvider()
	case HasProtoContextV2:
		return req.GetContextV2().GetProvider()
	case HasProtoContext:
		return req.GetContext().GetProvider()
	default:
		return ""
	}
}

func getDefaultProjectID(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
) (uuid.UUID, error) {
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
		return uuid.UUID{}, status.Errorf(codes.Internal, "cannot find projects for user: %v", err)
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
			DisplayName: authz.AllRolesDisplayName[role],
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
	invitations := make([]*minder.Invitation, 0)

	as, err := s.authzClient.AssignmentsToProject(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	if flags.Bool(ctx, s.featureFlags, flags.IDPResolver) {
		for i := range as {
			identity, err := s.idClient.Resolve(ctx, as[i].Subject)
			if err != nil {
				// if we can't resolve the subject, report the raw ID value
				zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
				continue
			}
			as[i].Subject = identity.Human()
		}
	}
	if flags.Bool(ctx, s.featureFlags, flags.UserManagement) {
		mapIdToDisplay := make(map[string]string, len(as))
		for i := range as {
			if mapIdToDisplay[as[i].Subject] == "" {
				user, err := s.idClient.Resolve(ctx, as[i].Subject)
				if err != nil {
					// if we can't resolve the subject, report the raw ID value
					zerolog.Ctx(ctx).Error().Err(err).Str("user", as[i].Subject).Msg("error resolving user")
					continue
				}
				mapIdToDisplay[as[i].Subject] = user.Human()
			}
			as[i].DisplayName = mapIdToDisplay[as[i].Subject]
		}
		// Add invitations, which are only stored in the Minder DB
		invites, err := s.store.ListInvitationsForProject(ctx, projectID)
		if err != nil {
			// return the information we can and log the error
			zerolog.Ctx(ctx).Error().Err(err).Msg("error getting invitations")
		}
		for _, invite := range invites {
			// nolint
			invitations = append(invitations, &minder.Invitation{
				Role:           invite.Role,
				Email:          invite.Email,
				Project:        projectID.String(),
				CreatedAt:      timestamppb.New(invite.CreatedAt),
				ExpiresAt:      timestamppb.New(invite.UpdatedAt.Add(7 * 24 * time.Hour)),
				Sponsor:        invite.IdentitySubject,
				SponsorDisplay: mapIdToDisplay[invite.IdentitySubject],
			})
		}
	}

	return &minder.ListRoleAssignmentsResponse{
		RoleAssignments: as,
		// TODO: for mocking purposes only, remove it when we implement the invitation flow
		// Invitations: invitations,
		Invitations: []*minder.Invitation{
			{
				Role:  "admin",
				Email: "bluey@sparkles.com",
				// TODO: project ID or name?
				Project:        projectID.String(),
				Code:           base64.StdEncoding.EncodeToString([]byte("0123456789")), // MDEyMzQ1Njc4OQ==
				Sponsor:        "5d9266d3-9bb7-4a20-b61c-019ca4bb75ac",
				SponsorDisplay: "Rusty Sparkles",
				CreatedAt:      &timestamppb.Timestamp{Seconds: time.Now().Unix()},
				ExpiresAt:      &timestamppb.Timestamp{Seconds: time.Now().Unix() + 36000},
			},
			{
				Role:           "permissions_manager",
				Email:          "bluey@sparkles.com",
				Project:        projectID.String(),
				Code:           base64.StdEncoding.EncodeToString([]byte("9876543210")), // OTg3NjU0MzIxMA==
				Sponsor:        "24f66e9f-fb3f-4c54-9e1a-e13d516b270d",
				SponsorDisplay: "Bingo Sparkles",
				CreatedAt:      &timestamppb.Timestamp{Seconds: time.Now().Unix()},
				ExpiresAt:      &timestamppb.Timestamp{Seconds: time.Now().Unix() + 36000},
			},
		},
	}, nil
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

	// TODO: for mocking purposes only, remove it when we implement the invitation flow
	// If the subject is an email, we can assume it's an invitation
	// TODO: Is there a problem that we reuse the subject?
	if isSubjectEmail(sub) {
		return &minder.AssignRoleResponse{
			// TODO: Should there be a role assignment when it's an invitation?
			Invitation: &minder.Invitation{
				Role:    role,
				Email:   sub,
				Project: engine.EntityFromContext(ctx).Project.ID.String(),
				// TODO: is the nonce format okay?
				Code: base64.StdEncoding.EncodeToString([]byte("0123456789")), // MDEyMzQ1Njc4OQ==
				// TODO: resolve to the user who triggered the assignment
				Sponsor:        "8efdfd4f-180e-4528-8bd7-dfe7f4aeff0a",
				SponsorDisplay: "Rusty Sparkles",
				CreatedAt:      &timestamppb.Timestamp{Seconds: time.Now().Unix()},
				ExpiresAt:      &timestamppb.Timestamp{Seconds: time.Now().Unix() + 604800}, // 1 week
			},
		}, nil
	}
	// We may be given a human-readable identifier which can vary over time. Resolve
	// it to an IDP-specific stable identifier so that we can support subject renames.
	identity := &auth.Identity{
		Provider:  nil,
		UserID:    sub,
		HumanName: sub,
	}
	if flags.Bool(ctx, s.featureFlags, flags.IDPResolver) {
		identity, err = s.idClient.Resolve(ctx, sub)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
			return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", sub)
		}
	}

	// Verify if user exists.
	// TODO: this assumes that we store all users in the database, and that we don't
	// need to namespace identify providers.  We should revisit these assumptions.
	//
	// Note: We could use `identity.String()` here, relying on Keycloak being registered
	// as the default with Provider.String() == "".
	if _, err := s.store.GetUserBySubject(ctx, identity.UserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "User not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
	}

	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	if err := s.authzClient.Write(ctx, identity.String(), authzrole, projectID); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := projectID.String()
	return &minder.AssignRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role,
			Subject: identity.Human(),
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

	// We may be given a human-readable identifier which can vary over time. Resolve
	// it to an IDP-specific stable identifier so that we can support subject renames.
	identity := &auth.Identity{
		Provider:  nil,
		UserID:    sub,
		HumanName: sub,
	}
	if flags.Bool(ctx, s.featureFlags, flags.IDPResolver) {
		identity, err = s.idClient.Resolve(ctx, sub)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
			return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", sub)
		}
	}

	// Verify if user exists
	if _, err := s.store.GetUserBySubject(ctx, identity.UserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "User not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
	}

	// Determine target project.
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	if err := s.authzClient.Delete(ctx, identity.String(), authzrole, projectID); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := projectID.String()
	return &minder.RemoveRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role,
			Subject: identity.Human(),
			Project: &respProj,
		},
	}, nil
}

// isSubjectEmail checks if the subject is an email address or not
func isSubjectEmail(subject string) bool {
	// Define the regular expression for validating an email address
	const emailRegexPattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	emailRegex := regexp.MustCompile(emailRegexPattern)
	return emailRegex.MatchString(subject)
}
