// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"path"
	"strconv"

	"github.com/google/uuid"
	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/projects"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// CreateUser is a service for user self registration
func (s *Server) CreateUser(ctx context.Context,
	_ *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {

	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	token, err := s.jwt.ParseAndValidate(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction")
	}
	defer s.store.Rollback(tx)
	qtx := s.store.GetQuerierWithTransaction(tx)
	if qtx == nil {
		return nil, status.Errorf(codes.Internal, "failed to get transaction")
	}

	subject := token.Subject()

	user, err := qtx.CreateUser(ctx, subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}
	var userProjects []*db.Project

	userProjects = append(userProjects, s.claimGitHubInstalls(ctx, qtx)...)

	if len(userProjects) == 0 {
		// Set up the default project for the user
		baseName := subject
		if token.PreferredUsername() != "" {
			// Ensure there's no existing project with that name. In case there is, we will append a
			// random string to the project name. This is a temporary solution until we have a better
			// way to handle this, i.e. this should happen separately from user creation.
			baseName, err = getUniqueProjectBaseName(ctx, s.store, token.PreferredUsername())
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get unique project name: %s", err)
			}
		}

		project, err := s.projectCreator.ProvisionSelfEnrolledProject(
			ctx,
			qtx,
			baseName,
			subject,
		)
		if err != nil {
			if errors.Is(err, projects.ErrProjectAlreadyExists) {
				return nil, util.UserVisibleError(codes.AlreadyExists, "project named %s already exists", baseName)
			}
			return nil, status.Errorf(codes.Internal, "failed to create default organization records: %s", err)
		}
		userProjects = append(userProjects, project)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = userProjects[0].ID

	err = s.store.Commit(tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %s", err)
	}

	if len(userProjects) == 0 {
		return nil, status.Errorf(codes.Internal, "failed to create any projects for user")
	}

	return &pb.CreateUserResponse{
		Id:              user.ID,
		ProjectId:       userProjects[0].ID.String(),
		ProjectName:     userProjects[0].Name,
		IdentitySubject: user.IdentitySubject,
		CreatedAt:       timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *Server) claimGitHubInstalls(ctx context.Context, qtx db.ExtendQuerier) []*db.Project {
	ghId, ok := jwt.GetUserClaimFromContext[string](ctx, "gh_id")
	if !ok || ghId == "" {
		return nil
	}
	installs, err := qtx.GetUnclaimedInstallationsByUser(ctx, sql.NullString{String: ghId, Valid: true})
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("gh_id", ghId).Msg("failed to get unclaimed installations")
		return nil
	}

	userID, err := strconv.ParseInt(ghId, 10, 64)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("gh_id", ghId).Msg("failed to parse gh_id")
		return nil
	}

	var userProjects []*db.Project

	for _, i := range installs {
		// TODO: if we can get an GitHub auth token for the user, we can do the rest with CreateGitHubAppWithoutInvitation
		proj, err := s.ghProviders.CreateGitHubAppWithoutInvitation(ctx, qtx, userID, i.AppInstallationID)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Int64("org_id", i.OrganizationID).Msg("failed to create GitHub app at first login")
			continue
		}
		if proj != nil {
			userProjects = append(userProjects, proj)
		}
	}

	return userProjects
}

// DeleteUser is a service for user self deletion
func (s *Server) DeleteUser(ctx context.Context,
	_ *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {

	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	token, err := s.jwt.ParseAndValidate(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}

	subject := token.Subject()

	err = DeleteUser(ctx, s.store, s.authzClient, s.projectDeleter, subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user from database: %v", err)
	}

	resp, err := s.cfg.Identity.Server.AdminDo(ctx, "DELETE", path.Join("users", subject), nil, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete account on IdP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return nil, status.Errorf(codes.Internal, "unexpected status code when deleting account: %d", resp.StatusCode)
	}

	return &pb.DeleteUserResponse{}, nil
}

func (s *Server) getUserDependencies(ctx context.Context, user db.User) ([]*pb.ProjectRole, []*pb.Project, error) {
	// get all the projects associated with that user
	projs, err := s.authzClient.ProjectsForUser(ctx, user.IdentitySubject)
	if err != nil {
		return nil, nil, err
	}

	var projectRoles []*pb.ProjectRole
	var deprecatedPrjs []*pb.Project
	for _, proj := range projs {
		pinfo, err := s.store.GetProjectByID(ctx, proj)
		if err != nil {
			// if the project was deleted while iterating, skip it
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, nil, err
		}

		// Try to parse the project metadata to complete the response fields
		pDisplay := pinfo.Name
		pDescr := ""
		meta, err := projects.ParseMetadata(&pinfo)
		if err == nil {
			pDisplay = meta.Public.DisplayName
			pDescr = meta.Public.Description
		}

		// Get all role assignments for this project
		as, err := s.authzClient.AssignmentsToProject(ctx, proj)
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
		}

		// TODO: Delete once all use ProjectRoles
		deprecatedPrjs = append(deprecatedPrjs, &pb.Project{
			ProjectId:   proj.String(),
			Name:        pinfo.Name,
			CreatedAt:   timestamppb.New(pinfo.CreatedAt),
			UpdatedAt:   timestamppb.New(pinfo.UpdatedAt),
			DisplayName: pDisplay,
			Description: pDescr,
		})

		var projectRole *pb.Role
		if len(as) != 0 {
			// Find the role for the user
			var roleString string
			for _, a := range as {
				if a.Subject == user.IdentitySubject {
					roleString = a.Role
				}
			}
			// Parse role
			authzRole, err := authz.ParseRole(roleString)
			if err != nil {
				return nil, nil, status.Errorf(codes.Internal, "failed to parse role: %v", err)
			}
			projectRole = &pb.Role{
				Name:        authzRole.String(),
				DisplayName: authz.AllRolesDisplayName[authzRole],
				Description: authz.AllRolesDescriptions[authzRole],
			}
		}

		// Append the project role to the response
		projectRoles = append(projectRoles, &pb.ProjectRole{
			Role: projectRole,
			Project: &pb.Project{
				ProjectId:   proj.String(),
				Name:        pinfo.Name,
				CreatedAt:   timestamppb.New(pinfo.CreatedAt),
				UpdatedAt:   timestamppb.New(pinfo.UpdatedAt),
				DisplayName: pDisplay,
				Description: pDescr,
			},
		})
	}

	return projectRoles, deprecatedPrjs, nil
}

// GetUser is a service for getting personal user details
func (s *Server) GetUser(ctx context.Context, _ *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// user is always authorized to get themselves
	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	openIdToken, err := s.jwt.ParseAndValidate(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}

	// check if user exists
	user, err := s.store.GetUserBySubject(ctx, openIdToken.Subject())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	var resp pb.GetUserResponse
	resp.User = &pb.UserRecord{
		Id:              user.ID,
		IdentitySubject: user.IdentitySubject,
		CreatedAt:       timestamppb.New(user.CreatedAt),
		UpdatedAt:       timestamppb.New(user.UpdatedAt),
	}

	projectRoles, deprecatedPrjs, err := s.getUserDependencies(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}
	resp.ProjectRoles = projectRoles
	// nolint: staticcheck
	resp.Projects = deprecatedPrjs
	return &resp, nil
}

// ListInvitations is a service for listing invitations.
func (s *Server) ListInvitations(ctx context.Context, _ *pb.ListInvitationsRequest) (*pb.ListInvitationsResponse, error) {
	invitations, err := s.invites.GetInvitesForSelf(ctx, s.store, s.idClient)
	if err != nil {
		return nil, err
	}

	return &pb.ListInvitationsResponse{
		Invitations: invitations,
	}, nil
}

// ResolveInvitation is a service for resolving an invitation.
func (s *Server) ResolveInvitation(ctx context.Context, req *pb.ResolveInvitationRequest) (*pb.ResolveInvitationResponse, error) {
	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction")
	}
	defer s.store.Rollback(tx)
	qtx := s.store.GetQuerierWithTransaction(tx)
	if qtx == nil {
		return nil, status.Errorf(codes.Internal, "failed to get transaction")
	}

	invite, err := s.invites.GetInvite(ctx, qtx, req.Code)
	if err != nil || invite == nil {
		return nil, err
	}
	project, err := uuid.Parse(invite.GetProject())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse project ID: %s", err)
	}

	// Accept invitation
	if req.Accept {
		_, err := ensureUser(ctx, s, qtx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get or create user: %s", err)
		}
		if err := s.acceptInvitation(ctx, project, invite); err != nil {
			return nil, err
		}
	}

	// Delete the invitation since its resolved
	err = s.invites.RemoveInvite(ctx, qtx, req.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete invitation: %s", err)
	}

	// Resolve the project's display name
	targetProject, err := qtx.GetProjectByID(ctx, project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project: %s", err)
	}

	err = s.store.Commit(tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %s", err)
	}

	// Parse the project metadata, so we can get the display name set by project owner
	meta, err := projects.ParseMetadata(&targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error parsing project metadata: %v", err)
	}

	return &pb.ResolveInvitationResponse{
		Role:           invite.GetRole(),
		Project:        invite.GetProject(),
		ProjectDisplay: meta.Public.DisplayName,
		Email:          invite.GetEmail(),
		IsAccepted:     req.Accept,
	}, nil
}

func (s *Server) acceptInvitation(ctx context.Context, projectID uuid.UUID, userInvite *pb.Invitation) error {
	// Validate in case there's an existing role assignment for the user
	as, err := s.authzClient.AssignmentsToProject(ctx, projectID)
	if err != nil {
		return status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}
	// Loop through all role assignments for the project and check if this user already has a role
	currentUser := auth.IdentityFromContext(ctx).String()
	for _, existing := range as {
		if existing.Subject == currentUser {
			// User already has the same role in the project
			if existing.Role == userInvite.Role {
				return util.UserVisibleError(codes.AlreadyExists, "user already has the same role in the project")
			}
			// Revoke the existing role assignments for the user in the project
			existingRole, err := authz.ParseRole(existing.Role)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to parse existing role: %s", err)
			}
			existingProject, err := uuid.Parse(*existing.Project)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to parse existing project: %s", err)
			}
			// Delete the role assignment
			if err := s.authzClient.Delete(
				ctx,
				currentUser,
				existingRole,
				existingProject,
			); err != nil {
				return status.Errorf(codes.Internal, "error writing role assignment: %v", err)
			}
		}
	}
	// Parse the role
	authzRole, err := authz.ParseRole(userInvite.Role)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to parse invitation role: %s", err)
	}
	// Add the user to the project
	if err := s.authzClient.Write(ctx, currentUser, authzRole, projectID); err != nil {
		return status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}
	return nil
}

func ensureUser(ctx context.Context, s *Server, store db.ExtendQuerier) (db.User, error) {
	id := auth.IdentityFromContext(ctx)
	sub := id.String()
	if sub == "" {
		return db.User{}, status.Error(codes.Internal, "failed to get user subject")
	}

	// Get the user by the subject
	user, err := store.GetUserBySubject(ctx, sub)
	if err == nil {
		return user, nil
	}

	// Create a user if necessary, see https://github.com/mindersec/minder/pull/3837/files#r1674108001
	if errors.Is(err, sql.ErrNoRows) {
		user, err := store.CreateUser(ctx, sub)
		if err != nil {
			return db.User{}, status.Errorf(codes.Internal, "failed to create user: %s", err)
		}
		// If we create a new user, we should see if they have outstanding GitHub installations
		// to create projects for.
		s.claimGitHubInstalls(ctx, store)
		return user, err
	}

	return db.User{}, err
}

// generateRandomString is used to generate a random string of a given length
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// getUniqueProjectBaseName is used to generate a unique project name
func getUniqueProjectBaseName(ctx context.Context, store db.Store, baseName string) (string, error) {
	const maxRetries = 10
	retryCount := 0
	uniqueBaseName := baseName
	for retryCount < maxRetries {
		_, err := store.GetProjectByName(ctx, uniqueBaseName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return uniqueBaseName, nil
			}
			return "", status.Errorf(codes.Internal, "failed to get project by name: %s", err)
		}
		r, err := generateRandomString(4)
		if err != nil {
			return "", status.Errorf(codes.Internal, "failed to generate random string: %s", err)
		}
		uniqueBaseName = baseName + "-" + r
		retryCount++
	}
	return "", status.Errorf(codes.ResourceExhausted, "failed to generate a unique project base name after %d attempts", maxRetries)
}
