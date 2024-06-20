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
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
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
	"github.com/stacklok/minder/internal/invite"
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
	invitations := make([]*minder.Invitation, 0)
	// Determine the target project.
	entityCtx := engine.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	as, err := s.authzClient.AssignmentsToProject(ctx, targetProject)
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
		invites, err := s.store.ListInvitationsForProject(ctx, targetProject)
		if err != nil {
			// return the information we can and log the error
			zerolog.Ctx(ctx).Error().Err(err).Msg("error getting invitations")
		}
		for _, i := range invites {
			invitations = append(invitations, &minder.Invitation{
				Role:           i.Role,
				Email:          i.Email,
				Project:        targetProject.String(),
				CreatedAt:      timestamppb.New(i.CreatedAt),
				ExpiresAt:      timestamppb.New(i.UpdatedAt.Add(7 * 24 * time.Hour)),
				Expired:        time.Now().After(i.UpdatedAt.Add(7 * 24 * time.Hour)),
				Sponsor:        i.IdentitySubject,
				SponsorDisplay: mapIdToDisplay[i.IdentitySubject],
			})
		}
	}

	return &minder.ListRoleAssignmentsResponse{
		RoleAssignments: as,
		Invitations:     invitations,
	}, nil
}

// AssignRole assigns a role to a user on a project.
// Note that this assumes that the request has already been authorized.
func (s *Server) AssignRole(ctx context.Context, req *minder.AssignRoleRequest) (*minder.AssignRoleResponse, error) {
	role := req.GetRoleAssignment().GetRole()
	sub := req.GetRoleAssignment().GetSubject()
	email := req.GetRoleAssignment().GetEmail()

	// Determine the target project.
	entityCtx := engine.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	// Ensure user is not updating their own role
	err := s.isUserSelfUpdating(ctx, sub, email)
	if err != nil {
		return nil, err
	}

	// Parse role (this also validates)
	authzRole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// Ensure the target project exists
	_, err = s.store.GetProjectByID(ctx, targetProject)
	if err != nil {
		// If the project is not found, return an error
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.InvalidArgument, "target project with ID %s not found", targetProject)
		}
		return nil, status.Errorf(codes.InvalidArgument, "error getting project: %v", err)
	}

	// Validate the subject and email - decide if it's an invitation or a role assignment
	if sub == "" && email != "" && isEmail(email) {
		if flags.Bool(ctx, s.featureFlags, flags.UserManagement) {
			return s.inviteUser(ctx, targetProject, authzRole, email)
		}
		return nil, util.UserVisibleError(codes.Unimplemented, "user management is not enabled")
	} else if sub != "" && email == "" {
		return s.assignRole(ctx, targetProject, authzRole, sub)
	}
	return nil, util.UserVisibleError(codes.InvalidArgument, "one of subject or email must be specified")
}

func (s *Server) inviteUser(
	ctx context.Context,
	targetProject uuid.UUID,
	role authz.Role,
	email string,
) (*minder.AssignRoleResponse, error) {
	var userInvite db.UserInvite
	// Current user is always authorized to get themselves
	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	openIdToken, err := s.jwt.ParseAndValidate(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}

	// Get the sponsor's user information (current user)
	currentUser, err := s.store.GetUserBySubject(ctx, openIdToken.Subject())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	// Check if the user is already invited
	existingInvite, err := s.store.GetInvitationByEmailAndProjectAndRole(ctx, db.GetInvitationByEmailAndProjectAndRoleParams{
		Email:   email,
		Project: targetProject,
		Role:    role.String(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// If there are no invitations for this email, great, we should create one
			userInvite, err = s.store.CreateInvitation(ctx, db.CreateInvitationParams{
				Code:    invite.GenerateCode(),
				Email:   email,
				Role:    role.String(),
				Project: targetProject,
				Sponsor: currentUser.ID,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "error creating invitation: %v", err)
			}
		} else {
			// Some other error happened, return it
			return nil, status.Errorf(codes.Internal, "error getting invitations: %v", err)
		}
	} else {
		// If we didn't get an error, this means there's an existing invite.
		// We should update its expiration and send the response.
		userInvite, err = s.store.UpdateInvitation(ctx, existingInvite.Code)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error updating invitation: %v", err)
		}
	}
	// If we are here, this means we either created a new invite or updated an existing one

	// Resolve the sponsor's identity and display name
	identity := &auth.Identity{
		Provider:  nil,
		UserID:    currentUser.IdentitySubject,
		HumanName: currentUser.IdentitySubject,
	}
	if flags.Bool(ctx, s.featureFlags, flags.IDPResolver) {
		identity, err = s.idClient.Resolve(ctx, currentUser.IdentitySubject)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
			return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", currentUser.IdentitySubject)
		}
	}

	// TODO: Publish the event for sending the invitation email

	// Send the invitation response
	return &minder.AssignRoleResponse{
		// Leaving the role assignment empty as it's an invitation
		Invitation: &minder.Invitation{
			Role:           userInvite.Role,
			Email:          userInvite.Email,
			Project:        userInvite.Project.String(),
			Code:           userInvite.Code,
			Sponsor:        identity.UserID,
			SponsorDisplay: identity.Human(),
			CreatedAt:      timestamppb.New(userInvite.CreatedAt),
			ExpiresAt:      timestamppb.New(userInvite.UpdatedAt.Add(7 * 24 * time.Hour)),
			Expired:        time.Now().After(userInvite.UpdatedAt.Add(7 * 24 * time.Hour)),
		},
	}, nil
}

func (s *Server) assignRole(
	ctx context.Context,
	targetPrj uuid.UUID,
	role authz.Role,
	subject string,
) (*minder.AssignRoleResponse, error) {
	var err error
	// We may be given a human-readable identifier which can vary over time. Resolve
	// it to an IDP-specific stable identifier so that we can support subject renames.
	identity := &auth.Identity{
		Provider:  nil,
		UserID:    subject,
		HumanName: subject,
	}
	if flags.Bool(ctx, s.featureFlags, flags.IDPResolver) {
		identity, err = s.idClient.Resolve(ctx, subject)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
			return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", subject)
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

	// Check in case there's an existing role assignment for the user
	as, err := s.authzClient.AssignmentsToProject(ctx, targetPrj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	for _, a := range as {
		if a.Subject == identity.String() {
			return nil, util.UserVisibleError(codes.AlreadyExists, "role assignment for this user already exists, use update instead")
		}
	}

	// Assign the role to the user
	if err := s.authzClient.Write(ctx, identity.String(), role, targetPrj); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := targetPrj.String()
	return &minder.AssignRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role.String(),
			Subject: identity.Human(),
			Project: &respProj,
		},
	}, nil
}

// RemoveRole removes a role from a user on a project
// Note that this assumes that the request has already been authorized.
func (s *Server) RemoveRole(ctx context.Context, req *minder.RemoveRoleRequest) (*minder.RemoveRoleResponse, error) {
	role := req.GetRoleAssignment().GetRole()
	sub := req.GetRoleAssignment().GetSubject()
	email := req.GetRoleAssignment().GetEmail()
	// Determine the target project.
	entityCtx := engine.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	// Ensure user is not updating their own role
	err := s.isUserSelfUpdating(ctx, sub, email)
	if err != nil {
		return nil, err
	}

	// Parse role (this also validates)
	authzRole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// Validate the subject and email - decide if it's about removing an invitation or a role assignment
	if sub == "" && email != "" && isEmail(email) {
		if flags.Bool(ctx, s.featureFlags, flags.UserManagement) {
			return s.removeInvite(ctx, targetProject, authzRole, email)
		}
		return nil, util.UserVisibleError(codes.Unimplemented, "user management is not enabled")
	} else if sub != "" && email == "" {
		// If there's a subject, we assume it's a role assignment
		return s.removeRole(ctx, targetProject, authzRole, sub)
	}
	return nil, util.UserVisibleError(codes.InvalidArgument, "one of subject or email must be specified")
}

func (s *Server) removeInvite(
	ctx context.Context,
	targetPrj uuid.UUID,
	role authz.Role,
	email string,
) (*minder.RemoveRoleResponse, error) {
	prj := targetPrj.String()
	// Get all invitations for this email, project and role
	inviteToRemove, err := s.store.GetInvitationByEmailAndProjectAndRole(ctx, db.GetInvitationByEmailAndProjectAndRoleParams{
		Email:   email,
		Project: targetPrj,
		Role:    role.String(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "no invitation found for this email, project and role")
		}
		return nil, status.Errorf(codes.Internal, "error getting invitation: %v", err)
	}

	// Delete the invitation
	_, err = s.store.DeleteInvitation(ctx, inviteToRemove.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error deleting invitation: %v", err)
	}

	// Return the response
	return &minder.RemoveRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role.String(),
			Email:   email,
			Project: &prj,
		},
	}, nil
}

func (s *Server) removeRole(
	ctx context.Context,
	targetProject uuid.UUID,
	role authz.Role,
	subject string,
) (*minder.RemoveRoleResponse, error) {
	var err error
	// We may be given a human-readable identifier which can vary over time. Resolve
	// it to an IDP-specific stable identifier so that we can support subject renames.
	identity := &auth.Identity{
		Provider:  nil,
		UserID:    subject,
		HumanName: subject,
	}
	if flags.Bool(ctx, s.featureFlags, flags.IDPResolver) {
		identity, err = s.idClient.Resolve(ctx, subject)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
			return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", subject)
		}
	}

	// Verify if user exists
	if _, err := s.store.GetUserBySubject(ctx, identity.UserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "User not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
	}

	if err := s.authzClient.Delete(ctx, identity.String(), role, targetProject); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}
	prj := targetProject.String()
	return &minder.RemoveRoleResponse{
		RoleAssignment: &minder.RoleAssignment{
			Role:    role.String(),
			Subject: identity.Human(),
			Project: &prj,
		},
	}, nil
}

// isEmail checks if the subject is an email address or not
func isEmail(subject string) bool {
	// Define the regular expression for validating an email address
	const emailRegexPattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	emailRegex := regexp.MustCompile(emailRegexPattern)
	return emailRegex.MatchString(subject)
}

// UpdateRole updates a role for a user on a project
func (s *Server) UpdateRole(ctx context.Context, req *minder.UpdateRoleRequest) (*minder.UpdateRoleResponse, error) {
	// For the time being, ensure only one role is updated at a time
	if len(req.GetRole()) != 1 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "only one role can be updated at a time")
	}
	role := req.GetRole()[0]
	sub := req.GetSubject()

	// Determine the target project.
	entityCtx := engine.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	if sub == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "role and subject must be specified")
	}

	// Ensure user is not updating their own role
	err := s.isUserSelfUpdating(ctx, sub, "")
	if err != nil {
		return nil, err
	}

	// Parse role (this also validates)
	authzRole, err := authz.ParseRole(role)
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

	// Remove the existing role assignment for the user
	as, err := s.authzClient.AssignmentsToProject(ctx, targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	for _, a := range as {
		if a.Subject == identity.String() {
			roleToDelete, err := authz.ParseRole(a.Role)
			if err != nil {
				return nil, util.UserVisibleError(codes.Internal, err.Error())
			}
			if err := s.authzClient.Delete(ctx, identity.String(), roleToDelete, targetProject); err != nil {
				return nil, status.Errorf(codes.Internal, "error deleting previous role assignment: %v", err)
			}
		}
	}

	// Update the role assignment for the user
	if err := s.authzClient.Write(ctx, identity.String(), authzRole, targetProject); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := targetProject.String()
	return &minder.UpdateRoleResponse{
		RoleAssignments: []*minder.RoleAssignment{
			{
				Role:    role,
				Subject: identity.Human(),
				Project: &respProj,
			},
		},
	}, nil
}

// isUserSelfUpdating is used to prevent if the user is trying to update their own role
func (s *Server) isUserSelfUpdating(ctx context.Context, subject, email string) error {
	// Ensure user is not updating their own role
	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}
	token, err := s.jwt.ParseAndValidate(tokenString)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}
	if subject != "" {
		if token.Subject() == subject {
			return util.UserVisibleError(codes.InvalidArgument, "cannot update your own role")
		}
	}
	if email != "" {
		if token.Email() == email {
			return util.UserVisibleError(codes.InvalidArgument, "cannot update your own role")
		}
	}
	return nil
}
