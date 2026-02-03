// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package repositories contains logic relating to the management of repos
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/models"
	"github.com/mindersec/minder/internal/entities/properties/service"
	entityService "github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/entities/service/validators"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/providers/manager"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// ErrRepoNotFound is returned when a repository is not found
var ErrRepoNotFound = errors.New("repository not found")

// RepositoryService encapsulates logic related to registering and deleting repos
// TODO: get rid of the github client from this interface
type RepositoryService interface {
	// CreateRepository registers a GitHub repository, including creating
	// a webhook in the repo in GitHub.
	CreateRepository(
		ctx context.Context,
		// TODO: this should just be ProviderID
		// Switch once we get rid of provider names from the repo table
		provider *db.Provider,
		projectID uuid.UUID,
		fetchByProps *properties.Properties,
	) (*pb.Repository, error)
	// DeleteByID removes the webhook and deletes the repo from the database.
	DeleteByID(
		ctx context.Context,
		repoID uuid.UUID,
		projectID uuid.UUID,
	) error
	// DeleteByName removes the webhook and deletes the repo from the database.
	// Ideally, we would take provider ID instead of name. Name is used for
	// backwards compatibility with the API endpoint which calls it.
	DeleteByName(
		ctx context.Context,
		repoOwner string,
		repoName string,
		projectID uuid.UUID,
		providerName string,
	) error

	// ListRepositories retrieves all repositories for the
	// specific provider and project.
	ListRepositories(
		ctx context.Context,
		projectID uuid.UUID,
		providerID uuid.UUID,
	) ([]*models.EntityWithProperties, error)

	// GetRepositoryById retrieves a repository by its ID and project.
	GetRepositoryById(ctx context.Context, repositoryID uuid.UUID, projectID uuid.UUID) (*pb.Repository, error)
	// GetRepositoryByName retrieves a repository by its name, owner, project and provider (if specified).
	GetRepositoryByName(
		ctx context.Context,
		repoOwner string,
		repoName string,
		projectID uuid.UUID,
		providerName string,
	) (*pb.Repository, error)
}

var (
	// ErrPrivateRepoForbidden is returned when creation fails due to an
	// attempt to register a private repo in a project which does not allow
	// private repos
	ErrPrivateRepoForbidden = validators.ErrPrivateRepoForbidden
	// ErrArchivedRepoForbidden is returned when creation fails due to an
	// attempt to register an archived repo
	ErrArchivedRepoForbidden = validators.ErrArchivedRepoForbidden
)

type repositoryService struct {
	store           db.Store
	eventProducer   interfaces.Publisher
	providerManager manager.ProviderManager
	propSvc         service.PropertiesService
	entityCreator   entityService.EntityCreator
}

// NewRepositoryService creates an instance of the RepositoryService interface
func NewRepositoryService(
	store db.Store,
	propSvc service.PropertiesService,
	eventProducer interfaces.Publisher,
	providerManager manager.ProviderManager,
	entityCreator entityService.EntityCreator,
) RepositoryService {
	return &repositoryService{
		store:           store,
		eventProducer:   eventProducer,
		providerManager: providerManager,
		propSvc:         propSvc,
		entityCreator:   entityCreator,
	}
}

func (r *repositoryService) CreateRepository(
	ctx context.Context,
	provider *db.Provider,
	projectID uuid.UUID,
	fetchByProps *properties.Properties,
) (*pb.Repository, error) {
	// Use the EntityCreator service to create the repository entity
	ewp, err := r.entityCreator.CreateEntity(ctx, provider, projectID,
		pb.Entity_ENTITY_REPOSITORIES, fetchByProps, &entityService.EntityCreationOptions{
			RegisterWithProvider:       true, // Create webhook
			PublishReconciliationEvent: true, // Publish reconciliation event
		})
	if err != nil {
		if errors.Is(err, validators.ErrPrivateRepoForbidden) ||
			errors.Is(err, validators.ErrArchivedRepoForbidden) {
			return nil, err
		}
		return nil, fmt.Errorf("error creating repository: %w", err)
	}

	// Convert to protobuf
	somePB, err := r.propSvc.EntityWithPropertiesAsProto(ctx, ewp, r.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	pbRepo, ok := somePB.(*pb.Repository)
	if !ok {
		return nil, fmt.Errorf("couldn't convert to protobuf. unexpected type: %T", somePB)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = provider.ID
	logger.BusinessRecord(ctx).Project = projectID
	logger.BusinessRecord(ctx).Repository = ewp.Entity.ID

	return pbRepo, nil
}

func (r *repositoryService) ListRepositories(
	ctx context.Context,
	projectID uuid.UUID,
	providerID uuid.UUID,
) (ents []*models.EntityWithProperties, outErr error) {
	tx, err := r.store.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}

	defer func() {
		if outErr != nil {
			if err := tx.Rollback(); err != nil {
				log.Printf("error rolling back transaction: %v", err)
			}
		}
	}()

	qtx := r.store.GetQuerierWithTransaction(tx)

	repoEnts, err := qtx.GetEntitiesByType(ctx, db.GetEntitiesByTypeParams{
		EntityType: db.EntitiesRepository,
		ProviderID: providerID,
		Projects:   []uuid.UUID{projectID},
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching repositories: %w", err)
	}

	ents = make([]*models.EntityWithProperties, 0, len(repoEnts))
	for _, ent := range repoEnts {
		ewp, err := r.propSvc.EntityWithPropertiesByID(ctx, ent.ID,
			service.CallBuilder().WithStoreOrTransaction(qtx))
		if err != nil {
			return nil, fmt.Errorf("error fetching properties for repository: %w", err)
		}

		if err := r.propSvc.RetrieveAllPropertiesForEntity(ctx, ewp, r.providerManager,
			service.ReadBuilder().WithStoreOrTransaction(qtx).TolerateStaleData()); err != nil {
			return nil, fmt.Errorf("error fetching properties for repository: %w", err)
		}

		ents = append(ents, ewp)
	}

	// We care about commiting the transaction since the `RetrieveAllPropertiesForEntity`
	// call above may have modified the properties of the entities
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return ents, nil
}

func (r *repositoryService) GetRepositoryById(
	ctx context.Context,
	repositoryID uuid.UUID,
	projectID uuid.UUID,
) (*pb.Repository, error) {
	ewp, err := r.propSvc.EntityWithPropertiesByID(ctx, repositoryID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching repository: %w", err)
	}

	// Verify the entity belongs to the correct project
	if ewp.Entity.ProjectID != projectID {
		return nil, sql.ErrNoRows
	}

	// Verify it's a repository entity
	if ewp.Entity.Type != pb.Entity_ENTITY_REPOSITORIES {
		return nil, fmt.Errorf("entity is not a repository")
	}

	// Retrieve all properties from provider
	if err := r.propSvc.RetrieveAllPropertiesForEntity(ctx, ewp, r.providerManager, nil); err != nil {
		return nil, fmt.Errorf("error fetching properties for repository: %w", err)
	}

	// Convert to protobuf
	somePB, err := r.propSvc.EntityWithPropertiesAsProto(ctx, ewp, r.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	pbRepo, ok := somePB.(*pb.Repository)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", somePB)
	}

	return pbRepo, nil
}

func (r *repositoryService) GetRepositoryByName(
	ctx context.Context,
	repoOwner string,
	repoName string,
	projectID uuid.UUID,
	providerName string,
) (*pb.Repository, error) {
	// Build the full repository name
	fullName := fmt.Sprintf("%s/%s", repoOwner, repoName)

	// Get provider ID from name if specified
	var providerID uuid.UUID
	if providerName != "" {
		prov, err := r.store.GetProviderByName(ctx, db.GetProviderByNameParams{
			Name:     providerName,
			Projects: []uuid.UUID{projectID},
		})
		if err != nil {
			return nil, fmt.Errorf("error fetching provider: %w", err)
		}
		providerID = prov.ID
	}

	// Search for repository by name property using V1 helper
	entities, err := r.store.GetTypedEntitiesByPropertyV1(
		ctx,
		db.EntitiesRepository,
		properties.PropertyName,
		fullName,
		db.GetTypedEntitiesOptions{
			ProjectID:  projectID,
			ProviderID: providerID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error searching for repository: %w", err)
	}

	if len(entities) == 0 {
		return nil, sql.ErrNoRows
	}

	// Use the first matching entity
	ewp, err := r.propSvc.EntityWithPropertiesByID(ctx, entities[0].ID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching repository: %w", err)
	}

	// Retrieve all properties from provider
	if err := r.propSvc.RetrieveAllPropertiesForEntity(ctx, ewp, r.providerManager, nil); err != nil {
		return nil, fmt.Errorf("error fetching properties for repository: %w", err)
	}

	// Convert to protobuf
	somePB, err := r.propSvc.EntityWithPropertiesAsProto(ctx, ewp, r.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	pbRepo, ok := somePB.(*pb.Repository)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", somePB)
	}

	return pbRepo, nil
}

func (r *repositoryService) DeleteByID(ctx context.Context, repositoryID uuid.UUID, projectID uuid.UUID) error {
	logger.BusinessRecord(ctx).Project = projectID
	logger.BusinessRecord(ctx).Repository = repositoryID

	ent, err := r.propSvc.EntityWithPropertiesByID(ctx, repositoryID, nil)
	if err != nil {
		return fmt.Errorf("error fetching repository: %w", err)
	}

	logger.BusinessRecord(ctx).ProviderID = ent.Entity.ProviderID

	prov, err := r.providerManager.InstantiateFromID(ctx, ent.Entity.ProviderID)
	if err != nil {
		return fmt.Errorf("error instantiating provider: %w", err)
	}

	return r.deleteRepository(ctx, prov, ent)
}

func (r *repositoryService) DeleteByName(
	ctx context.Context,
	repoOwner string,
	repoName string,
	projectID uuid.UUID,
	providerName string,
) error {
	logger.BusinessRecord(ctx).Project = projectID

	// Build the full repository name
	fullName := fmt.Sprintf("%s/%s", repoOwner, repoName)

	// Get provider ID from name if specified
	var providerID uuid.UUID
	if providerName != "" {
		prov, err := r.store.GetProviderByName(ctx, db.GetProviderByNameParams{
			Name:     providerName,
			Projects: []uuid.UUID{projectID},
		})
		if err != nil {
			return fmt.Errorf("error fetching provider: %w", err)
		}
		providerID = prov.ID
	}

	// Search for repository by name property using V1 helper
	entities, err := r.store.GetTypedEntitiesByPropertyV1(
		ctx,
		db.EntitiesRepository,
		properties.PropertyName,
		fullName,
		db.GetTypedEntitiesOptions{
			ProjectID:  projectID,
			ProviderID: providerID,
		},
	)
	if err != nil {
		return fmt.Errorf("error searching for repository: %w", err)
	}

	if len(entities) == 0 {
		return fmt.Errorf("error retrieving repository %s/%s in project %s: %w", repoOwner, repoName, projectID, sql.ErrNoRows)
	}

	// Fetch the entity with properties
	ent, err := r.propSvc.EntityWithPropertiesByID(ctx, entities[0].ID, nil)
	if err != nil {
		return fmt.Errorf("error fetching repository: %w", err)
	}

	logger.BusinessRecord(ctx).Repository = ent.Entity.ID

	prov, err := r.providerManager.InstantiateFromID(ctx, ent.Entity.ProviderID)
	if err != nil {
		return fmt.Errorf("error instantiating provider: %w", err)
	}

	return r.deleteRepository(ctx, prov, ent)
}

func (r *repositoryService) deleteRepository(
	ctx context.Context, client provifv1.Provider, repo *models.EntityWithProperties,
) error {
	var err error

	err = client.DeregisterEntity(ctx, pb.Entity_ENTITY_REPOSITORIES, repo.Properties)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Dict("properties", repo.Properties.ToLogDict()).
			Err(err).Msg("error deregistering repo")
	}

	_, err = db.WithTransaction(r.store, func(t db.ExtendQuerier) (*pb.Repository, error) {
		// Remove the entity from the DB
		if err := t.DeleteEntity(ctx, db.DeleteEntityParams{
			ID:        repo.Entity.ID,
			ProjectID: repo.Entity.ProjectID,
		}); err != nil {
			return nil, fmt.Errorf("error deleting entity from DB: %w", err)
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}

	return nil
}
