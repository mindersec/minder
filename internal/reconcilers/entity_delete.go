// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/entities/properties/service"
	minderlogger "github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

//nolint:exhaustive
func (r *Reconciler) handleEntityDeleteEvent(msg *message.Message) error {
	ctx := msg.Context()
	l := zerolog.Ctx(ctx).With().Logger()

	var event messages.MinderEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	// validate event
	validate := validator.New()
	if err := validate.Struct(&event); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		l.Error().Err(err).Msg("error validating event")
		return nil
	}

	if event.EntityType != pb.Entity_ENTITY_REPOSITORIES {
		l.Error().Str("entity_type", event.EntityType.String()).Msg("entity type not supported")
		return nil
	}

	l = zerolog.Ctx(ctx).With().
		Str("provider_id", event.ProviderID.String()).
		Str("project_id", event.ProjectID.String()).
		Str("entity_id", event.EntityID.String()).
		Logger()

	// Telemetry logging
	minderlogger.BusinessRecord(ctx).ProviderID = event.ProviderID
	minderlogger.BusinessRecord(ctx).Project = event.ProjectID

	l.Info().Msg("handling entity delete event")
	// Remove the entry in the DB. There's no need to clean any webhook we created for this repository, as GitHub
	// will automatically remove them when the repository is deleted.
	err := r.repos.DeleteByID(ctx, event.EntityID, event.ProjectID)
	if errors.Is(err, service.ErrEntityNotFound) {
		zerolog.Ctx(ctx).Debug().Err(err).
			Str("entity UUID", event.EntityID.String()).
			Msg("repository not found")
		return nil
	} else if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}

	minderlogger.BusinessRecord(ctx).Repository = event.EntityID
	return nil
}
