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
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/logger"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// nolint
func (r *Reconciler) handleEntityAddEvent(msg *message.Message) error {
	ctx := msg.Context()

	inf, err := entities.ParseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	l := zerolog.Ctx(ctx).With().
		Str("provider_id", inf.ProviderID.String()).
		Str("project_id", inf.ProjectID.String()).
		Str("entity_type", inf.Type.ToString()).
		Str("action", inf.ActionEvent).
		Logger()

	repoID, _, _ := inf.GetEntityDBIDs()

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = inf.ProviderID
	logger.BusinessRecord(ctx).Project = inf.ProjectID
	switch inf.Type {
	case pb.Entity_ENTITY_REPOSITORIES:
		l.Info().Str("repo_id", repoID.UUID.String()).Msg("handling entity add event")

		repoOwner := inf.GetRepositoryOwner()
		repoName := inf.GetRepositoryName()

		dbProvider, err := r.store.GetProviderByID(ctx, inf.ProviderID)
		if err != nil {
			return fmt.Errorf("error retrieving provider: %w", err)
		}

		if _, err := r.repos.CreateRepository(
			ctx,
			&dbProvider,
			inf.ProjectID,
			repoOwner,
			repoName,
		); err != nil {
			return fmt.Errorf("error add repository from DB: %w", err)
		}
		logger.BusinessRecord(ctx).Repository = repoID.UUID
		return nil
	default:
		err := fmt.Errorf("unsupported entity add event for: %s", inf.Type)
		l.Err(err).Msg("error handling entity add event")
		// Do not return the error, as we don't want to nack the message and retry
		return nil
	}
}
