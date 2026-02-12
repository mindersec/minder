// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/invites"
	"github.com/mindersec/minder/internal/util"
	minder "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

	relationName := relationAsName(relation)
	if relationName == "" {
		return nil, status.Errorf(codes.Internal, "error getting name for requested relation %v", relation)
	}

	entityCtx := engcontext.EntityFromContext(ctx)
	server := info.Server.(*Server)

	if err := server.authzClient.Check(ctx, relationName, entityCtx.Project.ID); err != nil {
		if errors.Is(err, authz.ErrNotAuthorized) && server.allowedAdminDelete(ctx, relation) {
			// Special case: permit deletions by admin users, but use the warning
			// log as an audit log.  (TODO: link this up with better audit logging
			// when that facility exists)
			zerolog.Ctx(ctx).Warn().Msgf("Permitting %s in %s by admin %s",
				relationName, entityCtx.Project.ID, auth.IdentityFromContext(ctx).String())
			return handler(ctx, req)
		}

		zerolog.Ctx(ctx).Error().Err(err).Msg("authorization check failed")
		return nil, util.UserVisibleError(
			codes.PermissionDenied, "user %q is not authorized to perform this operation on project %q",
			auth.IdentityFromContext(ctx).Human(), entityCtx.Project.ID)
	}

	return handler(ctx, req)
}

// relationAsName returns the OpenFGA relation name for the given relation enum.
// It returns an empty string in the case of error.
func relationAsName(relation minder.Relation) string {
	relationValue := relation.Descriptor().Values().ByNumber(relation.Number())
	if relationValue == nil {
		return ""
	}
	extension := proto.GetExtension(relationValue.Options(), minder.E_Name)
	relationName, ok := extension.(string)
	if !ok {
		return ""
	}
	return relationName
}

// populateEntityContext populates the project in the entity context, by looking at the proto context or
// fetching the default project
func populateEntityContext(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	req any,
) (context.Context, error) {
	projectID, err := getProjectIDFromRequest(ctx, req, store)
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

	entityCtx := &engcontext.EntityContext{
		Project: engcontext.Project{
			ID: projectID,
		},
		Provider: engcontext.Provider{
			Name: getProviderFromRequest(req),
		},
	}

	return engcontext.WithEntityContext(ctx, entityCtx), nil
}

func getProjectIDFromRequest(ctx context.Context, req any, store db.Store) (uuid.UUID, error) {
	var selectedProject string
	switch req := req.(type) {
	case HasProtoContextV2:
		selectedProject = req.GetContext().GetProjectId()
	case HasProtoContext:
		selectedProject = req.GetContext().GetProject()
	default:
		return uuid.Nil, status.Errorf(codes.Internal, "Error extracting context from request")
	}
	if selectedProject == "" || selectedProject == uuid.Nil.String() {
		return uuid.Nil, ErrNoProjectInContext
	}
	if id, err := uuid.Parse(selectedProject); err == nil {
		return id, nil
	}
	// We may have a string project name, look it up.
	project, err := store.GetProjectByName(ctx, selectedProject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, util.UserVisibleError(codes.InvalidArgument, "project %q not found", selectedProject)
		}
		return uuid.Nil, status.Errorf(codes.Internal, "error getting project: %v", err)
	}
	return project.ID, nil
}

func getProviderFromRequest(req any) string {
	switch req := req.(type) {
	case HasProtoContextV2:
		return req.GetContext().GetProvider()
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
	userId := auth.IdentityFromContext(ctx)

	// Not sure if we still need to do this at all, but we only create database users
	// for users registered in the primary ("") provider.
	if userId != nil && userId.String() == userId.UserID {
		_, err := store.GetUserBySubject(ctx, userId.String())
		if err != nil {
			// Note that we're revealing that the user is not registered in minder
			// since the caller has a valid token (this is checked in earlier middleware).
			// Therefore, we assume it's safe output that the user is not found.
			return uuid.UUID{}, util.UserVisibleError(codes.NotFound, "user not found")
		}
	}
	prjs, err := authzClient.ProjectsForUser(ctx, userId.String())
	if err != nil {
		return uuid.UUID{}, status.Errorf(codes.Internal, "cannot find projects for user: %v", err)
	}

	if len(prjs) == 0 {
		return uuid.UUID{}, util.UserVisibleError(codes.PermissionDenied,
			"user has no permissions in any projects.  Consider using CreateProject to create one.")
	}

	if len(prjs) != 1 {
		return uuid.UUID{}, util.UserVisibleError(codes.InvalidArgument, "Multiple projects found, cannot "+
			"determine default project. Please explicitly set a project and run the command again.")
	}

	return prjs[0], nil
}

func (s *Server) allowedAdminDelete(ctx context.Context, relation minder.Relation) bool {
	userId := auth.IdentityFromContext(ctx).String()

	adminIds := s.cfg.Authz.AdminDeleters
	deleteOps := []minder.Relation{
		minder.Relation_RELATION_DELETE,
		minder.Relation_RELATION_ROLE_ASSIGNMENT_REMOVE,
		minder.Relation_RELATION_REPO_DELETE,
		minder.Relation_RELATION_ARTIFACT_DELETE,
		minder.Relation_RELATION_PR_DELETE,
		minder.Relation_RELATION_PROVIDER_DELETE,
		minder.Relation_RELATION_RULE_TYPE_DELETE,
		minder.Relation_RELATION_PROFILE_DELETE,
		minder.Relation_RELATION_DATA_SOURCE_DELETE,
		minder.Relation_RELATION_ENTITY_DELETE,
	}
	return slices.Contains(adminIds, userId) && slices.Contains(deleteOps, relation)
}

// Permissions API
// ensure interface implementation
var _ minder.PermissionsServiceServer = (*Server)(nil)

// ListRoles returns the list of available roles for the minder instance
func (*Server) ListRoles(_ context.Context, _ *minder.ListRolesRequest) (*minder.ListRolesResponse, error) {
	resp := minder.ListRolesResponse{
		Roles: make([]*minder.Role, 0, len(authz.AllRolesDescriptions)),
	}
	// Iterate over all roles and add them to the response if they have a description. Skip if they don't.
	// The roles are sorted by the order in which they are defined in the authz package, i.e. admin, editor, viewer, etc.
	for _, role := range authz.AllRolesSorted {
		// Skip roles that don't have a description
		if authz.AllRolesDescriptions[role] == "" {
			continue
		}
		// Add the role to the response
		resp.Roles = append(resp.Roles, &minder.Role{
			Name:        role.String(),
			DisplayName: authz.AllRolesDisplayName[role],
			Description: authz.AllRolesDescriptions[role],
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
	entityCtx := engcontext.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	as, err := s.authzClient.AssignmentsToProject(ctx, targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	// Resolve the display names for the subjects
	mapIdToDisplay := make(map[string]string, len(as))
	for i := range as {
		identity, err := s.idClient.Resolve(ctx, as[i].Subject)
		if err != nil {
			// If we can't resolve the subject, report the raw ID value
			as[i].DisplayName = as[i].Subject
			if mapIdToDisplay[as[i].Subject] == "" {
				mapIdToDisplay[as[i].Subject] = as[i].Subject
			}
			zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
			continue
		}
		as[i].DisplayName = identity.Human()
		as[i].FirstName = identity.FirstName
		as[i].LastName = identity.LastName
		if mapIdToDisplay[as[i].Subject] == "" {
			mapIdToDisplay[as[i].Subject] = identity.Human()
		}
	}

	// Add invitations, which are only stored in the Minder DB
	projectInvites, err := s.store.ListInvitationsForProject(ctx, targetProject)
	if err != nil {
		// return the information we can and log the error
		zerolog.Ctx(ctx).Error().Err(err).Msg("error getting invitations")
	}
	for _, i := range projectInvites {
		invitations = append(invitations, &minder.Invitation{
			Role:           i.Role,
			Email:          i.Email,
			Project:        targetProject.String(),
			CreatedAt:      timestamppb.New(i.CreatedAt),
			ExpiresAt:      invites.GetExpireIn7Days(i.UpdatedAt),
			Expired:        invites.IsExpired(i.UpdatedAt),
			Sponsor:        i.IdentitySubject,
			SponsorDisplay: mapIdToDisplay[i.IdentitySubject],
			// Code is explicitly not returned here
		})
	}

	return &minder.ListRoleAssignmentsResponse{
		RoleAssignments: as,
		Invitations:     invitations,
	}, nil
}

// AssignRole assigns a role to a user on a project.
// Note that this assumes that the request has already been authorized.
//
//nolint:gocyclo  // There's a lot of trivial error handling here
func (s *Server) AssignRole(ctx context.Context, req *minder.AssignRoleRequest) (*minder.AssignRoleResponse, error) {
	role := req.GetRoleAssignment().GetRole()
	sub := req.GetRoleAssignment().GetSubject()
	inviteeEmail := req.GetRoleAssignment().GetEmail()

	// Determine the target project.
	entityCtx := engcontext.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	// Ensure user is not updating their own role
	err := isUserSelfUpdating(ctx, sub, inviteeEmail)
	if err != nil {
		return nil, err
	}

	// Parse role (this also validates)
	authzRole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "%s", err.Error())
	}

	// Ensure the target project exists
	_, err = s.store.GetProjectByID(ctx, targetProject)
	if err != nil {
		// If the project is not found, return an error
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.InvalidArgument, "target project with ID %s not found", targetProject)
		}
		return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
	}

	// Decide if it's an invitation or a role assignment
	if sub == "" && inviteeEmail != "" {
		invitation, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minder.Invitation, error) {
			return s.invites.CreateInvite(ctx, qtx, s.evt, s.cfg.Email, targetProject, authzRole, inviteeEmail)
		})
		if err != nil {
			return nil, err
		}

		return &minder.AssignRoleResponse{
			// Leaving the role assignment empty as it's an invitation
			Invitation: invitation,
		}, nil
	} else if sub != "" && inviteeEmail == "" {
		identity, err := s.idClient.Resolve(ctx, sub)
		if err != nil || identity == nil {
			return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", sub)
		}
		isMachine := identity.Provider.String() != ""
		if !isMachine {
			return nil, util.UserVisibleError(codes.Unimplemented, "human users may only be added by invitation")
		}
		assignment, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minder.RoleAssignment, error) {
			return s.roles.CreateRoleAssignment(ctx, qtx, s.authzClient, targetProject, *identity, authzRole)
		})
		if err != nil {
			return nil, err
		}

		return &minder.AssignRoleResponse{
			RoleAssignment: assignment,
		}, nil
	}
	return nil, util.UserVisibleError(codes.InvalidArgument, "one of subject or email must be specified")
}

// RemoveRole removes a role from a user on a project
// Note that this assumes that the request has already been authorized.
func (s *Server) RemoveRole(ctx context.Context, req *minder.RemoveRoleRequest) (*minder.RemoveRoleResponse, error) {
	role := req.GetRoleAssignment().GetRole()
	sub := req.GetRoleAssignment().GetSubject()
	inviteeEmail := req.GetRoleAssignment().GetEmail()
	// Determine the target project.
	entityCtx := engcontext.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	// Parse role (this also validates)
	authzRole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "%s", err.Error())
	}

	// Validate the subject and email - decide if it's about removing an invitation or a role assignment
	if sub == "" && inviteeEmail != "" {
		deletedInvitation, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minder.Invitation, error) {
			invites, err := s.invites.GetInvitesForEmail(ctx, qtx, targetProject, inviteeEmail)
			if err != nil {
				return nil, err
			}

			for _, i := range invites {
				if i.GetRole() == authzRole.String() {
					err := s.invites.RemoveInvite(ctx, qtx, i.GetCode())
					if err != nil {
						return nil, err
					}
					return i, nil
				}
			}

			return nil, util.UserVisibleError(codes.NotFound,
				"no invitation found for email %s with role %s in project %s",
				inviteeEmail, authzRole.String(), targetProject)
		})

		if err != nil {
			return nil, err
		}

		return &minder.RemoveRoleResponse{
			Invitation: deletedInvitation,
		}, nil
	} else if sub != "" && inviteeEmail == "" {
		// If there's a subject, we assume it's a role assignment
		deletedRoleAssignment, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minder.RoleAssignment, error) {
			return s.roles.RemoveRoleAssignment(ctx, qtx, s.authzClient, s.idClient, targetProject, sub, authzRole)
		})
		if err != nil {
			return nil, err
		}
		return &minder.RemoveRoleResponse{
			RoleAssignment: deletedRoleAssignment,
		}, nil
	}
	return nil, util.UserVisibleError(codes.InvalidArgument, "one of subject or email must be specified")
}

// UpdateRole updates a role for a user on a project
func (s *Server) UpdateRole(ctx context.Context, req *minder.UpdateRoleRequest) (*minder.UpdateRoleResponse, error) {
	// For the time being, ensure only one role is updated at a time
	if len(req.GetRoles()) != 1 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "only one role can be updated at a time")
	}
	role := req.GetRoles()[0]
	sub := req.GetSubject()
	inviteeEmail := req.GetEmail()

	// Determine the target project.
	entityCtx := engcontext.EntityFromContext(ctx)
	targetProject := entityCtx.Project.ID

	// Ensure user is not updating their own role
	err := isUserSelfUpdating(ctx, sub, inviteeEmail)
	if err != nil {
		return nil, err
	}

	// Parse role (this also validates)
	authzRole, err := authz.ParseRole(role)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "%s", err.Error())
	}

	// Validate the subject and email - decide if it's about updating an invitation or a role assignment
	if sub == "" && inviteeEmail != "" {
		updatedInvitation, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minder.Invitation, error) {
			return s.invites.UpdateInvite(ctx, qtx, s.evt, s.cfg.Email, targetProject, authzRole, inviteeEmail)
		})
		if err != nil {
			return nil, err
		}

		return &minder.UpdateRoleResponse{
			Invitations: []*minder.Invitation{
				updatedInvitation,
			},
		}, nil
	} else if sub != "" && inviteeEmail == "" {
		// If there's a subject, we assume it's a role assignment update
		updatedAssignment, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minder.RoleAssignment, error) {
			return s.roles.UpdateRoleAssignment(ctx, qtx, s.authzClient, s.idClient, targetProject, sub, authzRole)
		})
		if err != nil {
			return nil, err
		}

		return &minder.UpdateRoleResponse{
			RoleAssignments: []*minder.RoleAssignment{
				updatedAssignment,
			},
		}, nil
	}
	return nil, util.UserVisibleError(codes.InvalidArgument, "one of subject or email must be specified")
}

// isUserSelfUpdating is used to prevent if the user is trying to update their own role
func isUserSelfUpdating(ctx context.Context, subject, inviteeEmail string) error {
	if subject != "" {
		if auth.IdentityFromContext(ctx).String() == subject {
			return util.UserVisibleError(codes.InvalidArgument, "cannot update your own role")
		}
	}
	if inviteeEmail != "" {
		tokenEmail, err := jwt.GetUserEmailFromContext(ctx)
		if err != nil {
			return util.UserVisibleError(codes.Internal, "error getting user email from token: %v", err)
		}
		if tokenEmail == inviteeEmail {
			return util.UserVisibleError(codes.InvalidArgument, "cannot update your own role")
		}
	}
	return nil
}
