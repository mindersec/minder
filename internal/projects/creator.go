// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/marketplaces"
	github "github.com/stacklok/minder/internal/providers/github/clients"
	"github.com/stacklok/minder/pkg/mindpak"
)

// ProjectCreator encapsulates operations for managing projects
// TODO: There are several follow-ups needed here:
// 1. Move the delete operations into this interface
// 2. The interface is very GitHub-specific. It needs to be made more generic.
type ProjectCreator interface {
	// ProvisionSelfEnrolledOAuthProject creates the default records, such as projects,
	// roles and provider for the organization
	ProvisionSelfEnrolledOAuthProject(
		ctx context.Context,
		qtx db.Querier,
		projectName string,
		userSub string,
	) (outproj *db.Project, projerr error)

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

func (p *projectCreator) ProvisionSelfEnrolledOAuthProject(
	ctx context.Context,
	qtx db.Querier,
	projectName string,
	userSub string,
) (outproj *db.Project, projerr error) {
	project, err := p.ProvisionSelfEnrolledProject(ctx, qtx, projectName, userSub)
	if err != nil {
		return nil, err
	}

	// Create GitHub provider
	_, err = qtx.CreateProvider(ctx, db.CreateProviderParams{
		Name:       github.Github,
		ProjectID:  project.ID,
		Class:      db.ProviderClassGithub,
		Implements: github.OAuthImplements,
		Definition: json.RawMessage(`{"github": {}}`),
		AuthFlows:  github.OAuthAuthorizationFlows,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %v", err)
	}
	return project, nil
}

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
	if err := p.authzClient.Write(ctx, userSub, authz.AuthzRoleAdmin, projectID); err != nil {
		return nil, fmt.Errorf("failed to create authorization tuple: %w", err)
	}
	defer func() {
		// TODO: this can't be part of a transaction, so we should probably find a saga-ish
		// way to reverse this operation if the transaction fails.
		if outproj == nil && projerr != nil {
			if err := p.authzClient.Delete(ctx, userSub, authz.AuthzRoleAdmin, projectID); err != nil {
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
