// Copyright 2023 Stacklok, Inc
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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/reconcilers/messages"
)

//nolint:exhaustive
func (r *Reconciler) handleEntityDeleteEvent(msg *message.Message) error {
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
		return r.deleteGithubEntity(ctx, wcontext)
	// case db.ProviderClassGhcr:
	// case db.ProviderClassDockerhub:
	// case db.ProviderClassGitlab:
	default:
		return fmt.Errorf("unknown provider class: %s", dbProvider.Class)
	}
}

// NOTE: This should be moved to the github provider package.
func (r *Reconciler) deleteGithubEntity(
	ctx context.Context,
	wcontext *messages.CoreContext,
) error {
	// This switch statement should handle artifacts and pull
	// requests as well.
	switch wcontext.Type {
	case "repository":
		return r.deleteGithubRepository(ctx, wcontext)
	default:
		return fmt.Errorf("unknown entity type: %s", wcontext.Type)
	}
}

// NOTE: This should be moved to the github provider package.
func (r *Reconciler) deleteGithubRepository(
	ctx context.Context,
	wcontext *messages.CoreContext,
) error {
	var event messages.MinderEvent
	if err := json.Unmarshal(wcontext.Payload, &event); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	var repoIDStr string
	var ok bool
	if repoIDStr, ok = event.Entity["repoID"].(string); !ok {
		return errors.New("invalid repo id")
	}
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		return fmt.Errorf("invalid repo id: %w", err)
	}

	l := zerolog.Ctx(ctx).With().
		Str("provider_id", event.ProviderID.String()).
		Str("project_id", event.ProjectID.String()).
		Str("repo_id", repoID.String()).
		Logger()

	// validate event
	validate := validator.New()
	if err := validate.Struct(&event); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		l.Error().Err(err).Msg("error validating event")
		return nil
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = event.ProviderID
	logger.BusinessRecord(ctx).Project = event.ProjectID

	l.Info().Msg("handling entity delete event")
	// Remove the entry in the DB. There's no need to clean any webhook we created for this repository, as GitHub
	// will automatically remove them when the repository is deleted.
	if err := r.repos.DeleteByID(ctx, repoID, event.ProjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("error deleting repository from DB: %w", err)
	}

	logger.BusinessRecord(ctx).Repository = repoID
	return nil
}
