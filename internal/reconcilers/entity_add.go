// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconcilers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/reconcilers/messages"
)

// handleEntityAddEvent handles the entity add event.
// Although this method is meant to be generic and handle all types of entities,
// it currently only does so for repositories.
// Todo: Utilise for other entities when such are supported.
// nolint
func (r *Reconciler) handleEntityAddEvent(msg *message.Message) error {
	ctx := msg.Context()

	providerID, err := uuid.Parse(msg.Metadata.Get("providerID"))
	if err != nil {
		return fmt.Errorf("invalid provider id: %w", err)
	}
	projectID, err := uuid.Parse(msg.Metadata.Get("projectID"))
	if err != nil {
		return fmt.Errorf("invalid project id: %w", err)
	}
	wcontext := &messages.CoreContext{
		ProviderID: providerID,
		ProjectID:  projectID,
		Type:       msg.Metadata.Get("entityType"),
		Payload:    msg.Payload,
	}

	dbProvider, err := r.store.GetProviderByID(ctx, wcontext.ProviderID)
	if err != nil {
		return fmt.Errorf("error retrieving provider: %w", err)
	}

	switch dbProvider.Class {
	case db.ProviderClassGithub,
		db.ProviderClassGithubApp:
		// This should be a hook into provider-specific code.
		return r.AddGithubEntity(ctx, wcontext)
	// case db.ProviderClassGhcr:
	// case db.ProviderClassDockerhub:
	// case db.ProviderClassGitlab:
	default:
		return fmt.Errorf("unknown provider class: %s", dbProvider.Class)
	}
}

// AddGithubEntity adds a new entity to Minder.
//
// NOTE: This should be moved to the github provider package.
func (r *Reconciler) AddGithubEntity(
	ctx context.Context,
	wcontext *messages.CoreContext,
) error {
	// This switch statement should handle artifacts and pull
	// requests as well.
	switch wcontext.Type {
	case "repository":
		return r.addGithubRepository(ctx, wcontext)
	default:
		return fmt.Errorf("unknown entity type: %s", wcontext.Type)
	}
}

// NOTE: This should be moved to the github provider package.
func (r *Reconciler) addGithubRepository(
	ctx context.Context,
	wcontext *messages.CoreContext,
) error {
	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = wcontext.ProviderID
	logger.BusinessRecord(ctx).Project = wcontext.ProjectID

	var event messages.MinderEvent[*messages.RepoEvent]
	if err := json.Unmarshal(wcontext.Payload, &event); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	// validate event
	validate := validator.New()
	if err := validate.Struct(&event); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		l := zerolog.Ctx(ctx).With().
			Str("projectID", wcontext.ProjectID.String()).
			Logger()
		l.Error().Err(err).Msg("error validating event")
		return nil
	}

	dbProvider, err := r.store.GetProviderByID(ctx, wcontext.ProviderID)
	if err != nil {
		return fmt.Errorf("error retrieving provider: %w", err)
	}

	pbRepo, err := r.repos.CreateRepository(
		ctx,
		&dbProvider,
		event.ProjectID,
		event.Entity.RepoOwner,
		event.Entity.RepoName,
	)
	if err != nil {
		return fmt.Errorf("error add repository from DB: %w", err)
	}

	if pbRepo.Id == nil {
		return fmt.Errorf("repository id is nil")
	}
	repoID, err := uuid.Parse(*pbRepo.Id)
	if err != nil {
		return fmt.Errorf("repository id is not a UUID: %w", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Repository = repoID

	return nil
}
