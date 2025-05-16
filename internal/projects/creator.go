// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/marketplaces"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/mindpak"
)

// ProjectCreator encapsulates operations for managing projects
// TODO: There are several follow-ups needed here:
// 1. Move the delete operations into this interface
// 2. The interface is very GitHub-specific. It needs to be made more generic.
type ProjectCreator interface {

	// ProvisionSelfEnrolledProject creates the core default components of the project
	// (project, marketplace subscriptions, etc.).
	ProvisionSelfEnrolledProject(
		ctx context.Context,
		qtx db.ExtendQuerier,
		projectName string,
		userSub string,
	) (outproj *db.Project, projerr error)

	// ProvisionChildProject creates the components of a child project which is nested
	// under a parent in the project hierarchy.  The parent project is checked for
	// existence before creating the child project.
	ProvisionChildProject(
		ctx context.Context,
		qtx db.ExtendQuerier,
		parentProjectID uuid.UUID,
		projectName string,
	) (outproj *db.Project, projerr error)
}

type projectCreator struct {
	authzClient authz.Client
	marketplace marketplaces.Marketplace
	profilesCfg *server.DefaultProfilesConfig
	featuresCfg *server.FeaturesConfig
}

// NewProjectCreator creates a new instance of the project creator
func NewProjectCreator(authzClient authz.Client,
	marketplace marketplaces.Marketplace,
	profilesCfg *server.DefaultProfilesConfig,
	featuresCfg *server.FeaturesConfig,
) ProjectCreator {
	return &projectCreator{
		authzClient: authzClient,
		marketplace: marketplace,
		profilesCfg: profilesCfg,
		featuresCfg: featuresCfg,
	}
}

var (
	// ErrProjectAlreadyExists is returned when a project with the same name already exists
	ErrProjectAlreadyExists = errors.New("project already exists")
)

func (p *projectCreator) ProvisionSelfEnrolledProject(
	ctx context.Context,
	qtx db.ExtendQuerier,
	projectName string,
	userSub string,
) (outproj *db.Project, projerr error) {
	if err := ValidateName(projectName); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid project name: %s", err)
	}

	projectmeta := NewSelfEnrolledMetadata(projectName)

	jsonmeta, err := json.Marshal(&projectmeta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal meta: %w", err)
	}

	projectID := uuid.New()

	// Create authorization tuple
	if err := p.authzClient.Write(ctx, userSub, authz.RoleAdmin, projectID); err != nil {
		return nil, fmt.Errorf("failed to create authorization tuple: %w", err)
	}
	defer func() {
		// TODO: this can't be part of a transaction, so we should probably find a saga-ish
		// way to reverse this operation if the transaction fails.
		if outproj == nil && projerr != nil {
			if err := p.authzClient.Delete(ctx, userSub, authz.RoleAdmin, projectID); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to delete authorization tuple")
			}
		}
	}()

	// we need to create the default records for the organization
	project, err := qtx.CreateProjectWithID(ctx, db.CreateProjectWithIDParams{
		ID:       projectID,
		Name:     projectName,
		Metadata: jsonmeta,
	})
	if err != nil {
		// Check if `project_name_lower_idx` unique constraint was violated
		if db.ErrIsUniqueViolation(err) {
			return nil, ErrProjectAlreadyExists
		}
		return nil, fmt.Errorf("failed to create default project: %v", err)
	}

	// Retrieve the membership-to-feature mapping from the configuration
	projectFeatures := p.featuresCfg.GetFeaturesForMemberships(ctx)
	if err := qtx.CreateEntitlements(ctx, db.CreateEntitlementsParams{
		Features:  projectFeatures,
		ProjectID: project.ID,
	}); err != nil {
		return nil, fmt.Errorf("error creating entitlements: %w", err)
	}

	// Enable any default profiles and rule types in the project.
	// For now, we subscribe to a single bundle and a single profile.
	// Both are specified in the service config.
	bundleID := mindpak.ID(p.profilesCfg.Bundle.Namespace, p.profilesCfg.Bundle.Name)
	if err := p.marketplace.Subscribe(ctx, project.ID, bundleID, qtx); err != nil {
		return nil, fmt.Errorf("unable to subscribe to bundle: %w", err)
	}
	for _, profileName := range p.profilesCfg.GetProfiles() {
		if err := p.marketplace.AddProfile(ctx, project.ID, bundleID, profileName, qtx); err != nil {
			return nil, fmt.Errorf("unable to enable bundle profile: %w", err)
		}
	}

	return &project, nil
}

func (p *projectCreator) ProvisionChildProject(
	ctx context.Context,
	qtx db.ExtendQuerier,
	parentProjectID uuid.UUID,
	projectName string,
) (outproj *db.Project, projerr error) {
	parent, err := qtx.GetProjectByID(ctx, parentProjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
	}

	// Currently we only support one level of hierarchy, because the full hierarchy
	// has a bunch of complexities.
	// TODO: Remove this once we handle a full hierarchy
	if parent.ParentID.Valid {
		return nil, util.UserVisibleError(codes.InvalidArgument, "cannot create subproject of a subproject")
	}

	if err := ValidateName(projectName); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid project name: %v", err)
	}

	subProject, err := qtx.CreateProject(ctx, db.CreateProjectParams{
		Name: projectName,
		ParentID: uuid.NullUUID{
			UUID:  parentProjectID,
			Valid: true,
		},
		Metadata: json.RawMessage(`{}`),
	})
	if err != nil {
		if db.ErrIsUniqueViolation(err) {
			return nil, util.UserVisibleError(codes.AlreadyExists, "project named %s already exists", projectName)
		}
		return nil, status.Errorf(codes.Internal, "error creating subproject: %v", err)
	}

	// Retrieve the membership-to-feature mapping from the configuration
	projectFeatures := p.featuresCfg.GetFeaturesForMemberships(ctx)
	if err := qtx.CreateEntitlements(ctx, db.CreateEntitlementsParams{
		Features:  projectFeatures,
		ProjectID: subProject.ID,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "error creating entitlements: %v", err)
	}

	// We call "adopt" last here, because orchestrating the rollback of the Adopt is harder,
	// and we'd rather reduce the chances of leaking an adoption of a random UUID.
	if err := p.authzClient.Adopt(ctx, parent.ID, subProject.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "error creating subproject: %v", err)
	}

	return &subProject, nil
}
