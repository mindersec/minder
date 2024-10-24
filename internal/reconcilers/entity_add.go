// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/entities/properties"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/logger"
)

// handleEntityAddEvent handles the entity add event.
// Although this method is meant to be generic and handle all types of entities,
// it currently only does so for repositories.
// Todo: Utilise for other entities when such are supported.
// nolint
func (r *Reconciler) handleEntityAddEvent(msg *message.Message) error {
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
		l.Debug().Str("entity_type", event.EntityType.String()).Msg("unsupported entity type")
		return nil
	}

	if len(event.Properties) == 0 {
		zerolog.Ctx(ctx).Error().Msg("no properties in event")
		return nil
	}

	fetchByProps, err := properties.NewProperties(event.Properties)
	if err != nil {
		return fmt.Errorf("error creating properties: %w", err)
	}

	l = zerolog.Ctx(ctx).With().
		Str("provider_id", event.ProviderID.String()).
		Str("project_id", event.ProjectID.String()).
		Dict("properties", fetchByProps.ToLogDict()).
		Logger()

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = event.ProviderID
	logger.BusinessRecord(ctx).Project = event.ProjectID

	dbProvider, err := r.store.GetProviderByID(ctx, event.ProviderID)
	if err != nil {
		return fmt.Errorf("error retrieving provider: %w", err)
	}

	pbRepo, err := r.repos.CreateRepository(ctx, &dbProvider, event.ProjectID, fetchByProps)
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

	logger.BusinessRecord(ctx).Repository = repoID
	return nil
}
