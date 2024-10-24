// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/marketplaces"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/mindpak"
)

// ProjectCreator encapsulates operations for managing projects
// TODO: There are several follow-ups needed here:
// 1. Move the delete operations into this interface
// 2. The interface is very GitHub-specific. It needs to be made more generic.
type ProjectCreator interface {

	// ProvisionSelfEnrolledProject creates the core default components of the project
	// (project, marketplace subscriptions, etc.) but *does not* create a project.
	ProvisionSelfEnrolledProject(
		ctx context.Context,
		qtx db.Querier,
		projectName string,
		userSub string,
	) (outproj *db.Project, projerr error)
}

type projectCreator struct {
	authzClient authz.Client
	marketplace marketplaces.Marketplace
	profilesCfg *server.DefaultProfilesConfig
}

// NewProjectCreator creates a new instance of the project creator
func NewProjectCreator(authzClient authz.Client,
	marketplace marketplaces.Marketplace,
	profilesCfg *server.DefaultProfilesConfig,
) ProjectCreator {
	return &projectCreator{
		authzClient: authzClient,
		marketplace: marketplace,
		profilesCfg: profilesCfg,
	}
}

var (
	// ErrProjectAlreadyExists is returned when a project with the same name already exists
	ErrProjectAlreadyExists = errors.New("project already exists")
)

func (p *projectCreator) ProvisionSelfEnrolledProject(
	ctx context.Context,
	qtx db.Querier,
	projectName string,
	userSub string,
) (outproj *db.Project, projerr error) {
	if ValidateName(projectName) != nil {
		return nil, fmt.Errorf("invalid project name: %w", ErrValidationFailed)
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
