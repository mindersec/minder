// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service contains the service layer for entity creation
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/models"
	propService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers/manager"
	reconcilers "github.com/mindersec/minder/internal/reconcilers/messages"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// EntityCreationOptions configures entity creation behavior
type EntityCreationOptions struct {
	// Parent entity ID (for originated entities like artifacts, releases)
	OriginatingEntityID *uuid.UUID

	// Whether to register with provider (e.g., create webhooks)
	RegisterWithProvider bool

	// Whether to publish reconciliation events
	PublishReconciliationEvent bool
}

// EntityCreator creates entities in a consistent, reusable way
type EntityCreator interface {
	// CreateEntity creates an entity of any type
	CreateEntity(
		ctx context.Context,
		provider *db.Provider,
		projectID uuid.UUID,
		entityType pb.Entity,
		identifyingProps *properties.Properties,
		opts *EntityCreationOptions,
	) (*models.EntityWithProperties, error)
}

// EntityValidator validates entity creation based on business rules
type EntityValidator interface {
	// Validate returns nil if entity is valid, error otherwise
	Validate(
		ctx context.Context,
		entType pb.Entity,
		props *properties.Properties,
		projectID uuid.UUID,
	) error
}

type entityCreator struct {
	store           db.Store
	propSvc         propService.PropertiesService
	providerManager manager.ProviderManager
	eventProducer   interfaces.Publisher
	validators      []EntityValidator
}

// NewEntityCreator creates a new EntityCreator
func NewEntityCreator(
	store db.Store,
	propSvc propService.PropertiesService,
	providerManager manager.ProviderManager,
	eventProducer interfaces.Publisher,
	validators []EntityValidator,
) EntityCreator {
	return &entityCreator{
		store:           store,
		propSvc:         propSvc,
		providerManager: providerManager,
		eventProducer:   eventProducer,
		validators:      validators,
	}
}

func (e *entityCreator) CreateEntity(
	ctx context.Context,
	provider *db.Provider,
	projectID uuid.UUID,
	entityType pb.Entity,
	identifyingProps *properties.Properties,
	opts *EntityCreationOptions,
) (*models.EntityWithProperties, error) {
	// Default options if not provided
	if opts == nil {
		opts = &EntityCreationOptions{
			RegisterWithProvider:       entityType == pb.Entity_ENTITY_REPOSITORIES,
			PublishReconciliationEvent: entityType == pb.Entity_ENTITY_REPOSITORIES,
		}
	}

	// 1. Instantiate provider
	prov, err := e.providerManager.InstantiateFromID(ctx, provider.ID)
	if err != nil {
		return nil, fmt.Errorf("error instantiating provider: %w", err)
	}

	// 2. Check if provider supports this entity type
	if !prov.SupportsEntity(entityType) {
		return nil, fmt.Errorf("provider %s does not support entity type %s",
			provider.Name, entityType)
	}

	// 3. Fetch all properties from provider
	allProps, err := prov.FetchAllProperties(ctx, identifyingProps, entityType, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching properties: %w", err)
	}

	// 4. Run validators
	if err := e.runValidators(ctx, entityType, allProps, projectID); err != nil {
		return nil, err
	}

	// 5. Get entity name
	entityName, err := prov.GetEntityName(entityType, allProps)
	if err != nil {
		return nil, fmt.Errorf("error getting entity name: %w", err)
	}

	// 6. Register with provider if needed (e.g., create webhook)
	var registeredProps *properties.Properties
	if opts.RegisterWithProvider {
		registeredProps, err = prov.RegisterEntity(ctx, entityType, allProps)
		if err != nil {
			return nil, fmt.Errorf("error registering with provider: %w", err)
		}
	} else {
		registeredProps = allProps
	}

	// 7. Persist to database in transaction
	ewp, err := db.WithTransaction(e.store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
		// Generate entity ID
		entityID := uuid.New()

		// Prepare params
		params := db.CreateOrEnsureEntityByIDParams{
			ID:         entityID,
			EntityType: entities.EntityTypeToDB(entityType),
			Name:       entityName,
			ProjectID:  projectID,
			ProviderID: provider.ID,
		}

		// If this is an originating entity, set the originated_from field
		if opts.OriginatingEntityID != nil {
			params.OriginatedFrom = uuid.NullUUID{
				UUID:  *opts.OriginatingEntityID,
				Valid: true,
			}
		}

		// Create entity instance
		ent, err := t.CreateOrEnsureEntityByID(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("error creating entity: %w", err)
		}

		// Replace properties - use Replace to ensure a clean slate
		// (removes any stale properties from previous failed attempts)
		if err := e.propSvc.ReplaceAllProperties(ctx, ent.ID, registeredProps,
			propService.CallBuilder().WithStoreOrTransaction(t)); err != nil {
			return nil, fmt.Errorf("error saving properties: %w", err)
		}

		return models.NewEntityWithProperties(ent, registeredProps), nil
	})
	if err != nil {
		// Cleanup: Try to deregister from provider if we registered
		e.cleanupProviderRegistration(ctx, prov, entityType, registeredProps, opts.RegisterWithProvider)
		return nil, err
	}

	// 8. Publish reconciliation event if needed
	if opts.PublishReconciliationEvent {
		if err := e.publishReconciliationEvent(ctx, ewp, projectID, provider.ID); err != nil {
			// Log but don't fail - event publishing is non-critical
			zerolog.Ctx(ctx).Error().Err(err).
				Msg("error publishing reconciliation event")
		}
	}

	return ewp, nil
}

func (e *entityCreator) runValidators(
	ctx context.Context,
	entityType pb.Entity,
	allProps *properties.Properties,
	projectID uuid.UUID,
) error {
	for _, validator := range e.validators {
		if err := validator.Validate(ctx, entityType, allProps, projectID); err != nil {
			return err
		}
	}
	return nil
}

func (e *entityCreator) publishReconciliationEvent(
	_ context.Context,
	ewp *models.EntityWithProperties,
	projectID uuid.UUID,
	providerID uuid.UUID,
) error {
	// For now, only repositories have reconciliation events
	if ewp.Entity.Type != pb.Entity_ENTITY_REPOSITORIES {
		return nil
	}

	msg, err := reconcilers.NewRepoReconcilerMessage(providerID, ewp.Entity.ID, projectID)
	if err != nil {
		return fmt.Errorf("error creating reconciler message: %w", err)
	}

	if err := e.eventProducer.Publish(constants.TopicQueueReconcileRepoInit, msg); err != nil {
		return fmt.Errorf("error publishing reconciler event: %w", err)
	}

	return nil
}

func (*entityCreator) cleanupProviderRegistration(
	ctx context.Context,
	prov provifv1.Provider,
	entityType pb.Entity,
	registeredProps *properties.Properties,
	wasRegistered bool,
) {
	if !wasRegistered || registeredProps == nil {
		return
	}

	// Use background context for cleanup to avoid cancellation issues
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cleanupErr := prov.DeregisterEntity(cleanupCtx, entityType, registeredProps)
	if cleanupErr != nil {
		zerolog.Ctx(ctx).Error().Err(cleanupErr).
			Msg("error cleaning up provider registration after failure")
	}
}
