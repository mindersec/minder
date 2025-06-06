// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package roles contains the logic for managing user roles within a Minder project
package roles

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// RoleService encapsulates the methods to manage user role assignments
type RoleService interface {
	// CreateRoleAssignment assigns a user a role on a project
	CreateRoleAssignment(ctx context.Context, qtx db.Querier, authzClient authz.Client,
		targetProject uuid.UUID, subject auth.Identity, authzRole authz.Role) (*pb.RoleAssignment, error)

	// UpdateRoleAssignment updates the users role on a project
	UpdateRoleAssignment(ctx context.Context, qtx db.Querier, authzClient authz.Client, idClient auth.Resolver,
		targetProject uuid.UUID, subject string, authzRole authz.Role) (*pb.RoleAssignment, error)

	// RemoveRoleAssignment removes the role assignment for the user on a project
	RemoveRoleAssignment(ctx context.Context, qtx db.Querier, authzClient authz.Client, idClient auth.Resolver,
		targetProject uuid.UUID, subject string, roleToRemove authz.Role) (*pb.RoleAssignment, error)
}

type roleService struct {
}

// NewRoleService creates a new instance of RoleService
func NewRoleService() RoleService {
	return &roleService{}
}

func (*roleService) CreateRoleAssignment(ctx context.Context, qtx db.Querier, authzClient authz.Client,
	targetProject uuid.UUID, identity auth.Identity, authzRole authz.Role) (*pb.RoleAssignment, error) {

	// For users in the primary (human) identity store, verify if user exists in our database.
	// TODO: this assumes that we store human users in the Minder database, in addition to the
	// identity provider and the auth store.  We should revisit this assumption.
	if identity.Provider.String() == "" {
		if _, err := qtx.GetUserBySubject(ctx, identity.String()); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "User not found")
			}
			return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
		}
	}

	// Check in case there's an existing role assignment for the user
	as, err := authzClient.AssignmentsToProject(ctx, targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	for _, a := range as {
		if a.Subject == identity.String() {
			return nil, util.UserVisibleError(codes.AlreadyExists, "role assignment for this user already exists, use update instead")
		}
	}

	// Assign the role to the user
	if err := authzClient.Write(ctx, identity.String(), authzRole, targetProject); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := targetProject.String()
	return &pb.RoleAssignment{
		Role:    authzRole.String(),
		Subject: identity.Human(),
		Project: &respProj,
	}, nil
}

func (*roleService) UpdateRoleAssignment(ctx context.Context, qtx db.Querier, authzClient authz.Client,
	idClient auth.Resolver, targetProject uuid.UUID, sub string, authzRole authz.Role) (*pb.RoleAssignment, error) {
	// Resolve the subject to an identity
	identity, err := idClient.Resolve(ctx, sub)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
		return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", sub)
	}

	// Verify if user exists
	if identity.Provider.String() == "" {
		if _, err := qtx.GetUserBySubject(ctx, identity.String()); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "User not found")
			}
			return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
		}
	}

	// Remove the existing role assignment for the user
	as, err := authzClient.AssignmentsToProject(ctx, targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	for _, a := range as {
		if a.Subject == identity.String() {
			roleToDelete, err := authz.ParseRole(a.Role)
			if err != nil {
				return nil, util.UserVisibleError(codes.Internal, "%s", err.Error())
			}
			if err := authzClient.Delete(ctx, identity.String(), roleToDelete, targetProject); err != nil {
				return nil, status.Errorf(codes.Internal, "error deleting previous role assignment: %v", err)
			}
		}
	}

	// Update the role assignment for the user
	if err := authzClient.Write(ctx, identity.String(), authzRole, targetProject); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}

	respProj := targetProject.String()
	return &pb.RoleAssignment{
		Role:    authzRole.String(),
		Subject: identity.UserID,
		Project: &respProj,
	}, nil
}

func (*roleService) RemoveRoleAssignment(ctx context.Context, qtx db.Querier, authzClient authz.Client,
	idClient auth.Resolver, targetProject uuid.UUID, subject string, roleToRemove authz.Role) (*pb.RoleAssignment, error) {

	// Resolve the subject to an identity
	identity, err := idClient.Resolve(ctx, subject)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
		return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", subject)
	}

	// Verify if user exists
	if identity.Provider.String() == "" {
		if _, err := qtx.GetUserBySubject(ctx, identity.String()); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "User not found")
			}
			return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
		}
	}

	// Get all role assignments for the project
	as, err := authzClient.AssignmentsToProject(ctx, targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}

	// Check if there is such role assignment for the user or the user is the last admin
	found := false
	adminRolesCnt := 0
	for _, a := range as {
		if a.Subject == identity.String() && a.Role == roleToRemove.String() {
			found = true
		}
		if a.Role == authz.RoleAdmin.String() {
			adminRolesCnt++
		}
	}

	// If there's no role assignment for the user, return an error
	if !found {
		return nil, util.UserVisibleError(codes.NotFound, "role assignment for this user does not exist")
	}

	// If there's only one admin role, return an error
	if roleToRemove == authz.RoleAdmin && adminRolesCnt <= 1 {
		return nil, util.UserVisibleError(codes.FailedPrecondition, "cannot remove the last admin from the project")
	}

	// Delete the role assignment
	if err := authzClient.Delete(ctx, identity.String(), roleToRemove, targetProject); err != nil {
		return nil, status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}
	prj := targetProject.String()
	return &pb.RoleAssignment{
		Role:    roleToRemove.String(),
		Subject: identity.Human(),
		Project: &prj,
	}, nil
}
