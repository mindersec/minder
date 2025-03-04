// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service contains the service layer for entity instances
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/models"
	propService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers/manager"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// ErrEntityNotFound is returned when an entity is not found
var ErrEntityNotFound = errors.New("entity not found")

// EntityService encapsulates logic related to entity instances
type EntityService interface {
	// ListEntities retrieves all entities for the specific project and provider
	ListEntities(
		ctx context.Context,
		projectID uuid.UUID,
		providerID uuid.UUID,
		entityType pb.Entity,
		cursor string,
		limit int64,
	) ([]*pb.EntityInstance, string, error)

	// GetEntityByID retrieves an entity by its ID
	GetEntityByID(
		ctx context.Context,
		entityID uuid.UUID,
		projectID uuid.UUID,
	) (*pb.EntityInstance, error)

	// GetEntityByName retrieves an entity by its name
	GetEntityByName(
		ctx context.Context,
		name string,
		projectID uuid.UUID,
		providerID uuid.UUID,
		entityType pb.Entity,
	) (*pb.EntityInstance, error)

	// DeleteEntityByID deletes an entity by its ID
	DeleteEntityByID(
		ctx context.Context,
		entityID uuid.UUID,
		projectID uuid.UUID,
	) error
}

type entityService struct {
	store           db.Store
	propSvc         propService.PropertiesService
	providerManager manager.ProviderManager
}

// NewEntityService creates a new instance of the EntityService interface
func NewEntityService(
	store db.Store,
	propSvc propService.PropertiesService,
	providerManager manager.ProviderManager,
) EntityService {
	return &entityService{
		store:           store,
		propSvc:         propSvc,
		providerManager: providerManager,
	}
}

//nolint:gocyclo // Let's refactor this later
func (s *entityService) ListEntities(
	ctx context.Context,
	projectID uuid.UUID,
	providerID uuid.UUID,
	entityType pb.Entity,
	cursor string,
	limit int64,
) ([]*pb.EntityInstance, string, error) {
	l := zerolog.Ctx(ctx).With().
		Str("projectID", projectID.String()).
		Str("providerID", providerID.String()).
		Str("entityType", entityType.String()).
		Logger()

	// Convert pb.Entity to db.Entities
	dbEntityType, err := entities.EntityTypeToDBType(entityType)
	if err != nil {
		return nil, "", fmt.Errorf("failed to convert entity type: %w", err)
	}

	// Parse cursor if provided
	if cursor != "" {
		_, err := uuid.Parse(cursor)
		if err != nil {
			return nil, "", util.UserVisibleError(codes.InvalidArgument, "invalid cursor format")
		}
		// TODO: Use lastEntityID for pagination
	}

	// Set up query parameters
	queryLimit := sql.NullInt64{Valid: false, Int64: 0}
	if limit > 0 {
		const maxFetchLimit = 100
		if limit > maxFetchLimit {
			return nil, "", util.UserVisibleError(codes.InvalidArgument, "limit too high, max is %d", maxFetchLimit)
		}
		queryLimit = sql.NullInt64{Valid: true, Int64: limit + 1} // +1 to check if there are more results
	}

	// Get entities from database
	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, "", fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			l.Error().Err(err).Msg("error rolling back transaction")
		}
	}()

	qtx := s.store.GetQuerierWithTransaction(tx)

	// TODO: Implement proper pagination with cursor
	outentities, err := qtx.GetEntitiesByType(ctx, db.GetEntitiesByTypeParams{
		EntityType: dbEntityType,
		ProviderID: providerID,
		Projects:   []uuid.UUID{projectID},
	})
	if err != nil {
		return nil, "", fmt.Errorf("error fetching entities: %w", err)
	}

	// Convert to EntityWithProperties and fetch properties
	var results []*pb.EntityInstance
	var nextCursor string

	for i, entity := range outentities {
		// Apply limit if specified
		if queryLimit.Valid && int64(i) >= queryLimit.Int64-1 {
			nextCursor = entity.ID.String()
			break
		}

		ewp, err := s.propSvc.EntityWithPropertiesByID(ctx, entity.ID,
			propService.CallBuilder().WithStoreOrTransaction(qtx))
		if err != nil {
			return nil, "", fmt.Errorf("error fetching properties for entity: %w", err)
		}

		if err := s.propSvc.RetrieveAllPropertiesForEntity(ctx, ewp, s.providerManager,
			propService.ReadBuilder().WithStoreOrTransaction(qtx).TolerateStaleData()); err != nil {
			return nil, "", fmt.Errorf("error fetching properties for entity: %w", err)
		}

		// Convert to protobuf
		pbEntity := entityInstanceToProto(ewp)

		results = append(results, pbEntity)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("error committing transaction: %w", err)
	}

	return results, nextCursor, nil
}

func (s *entityService) GetEntityByID(
	ctx context.Context,
	entityID uuid.UUID,
	projectID uuid.UUID,
) (*pb.EntityInstance, error) {
	// Get entity from database
	entity, err := s.store.GetEntityByID(ctx, entityID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "entity not found")
		}
		return nil, fmt.Errorf("error fetching entity: %w", err)
	}

	// Verify entity belongs to the project
	if entity.ProjectID != projectID {
		return nil, status.Errorf(codes.NotFound, "entity not found in project")
	}

	// Get properties
	ewp, err := s.propSvc.EntityWithPropertiesByID(ctx, entityID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching properties for entity: %w", err)
	}

	if err := s.propSvc.RetrieveAllPropertiesForEntity(ctx, ewp, s.providerManager,
		propService.ReadBuilder().TolerateStaleData()); err != nil {
		return nil, fmt.Errorf("error fetching properties for entity: %w", err)
	}

	// Convert to protobuf
	return entityInstanceToProto(ewp), nil
}

func (s *entityService) GetEntityByName(
	ctx context.Context,
	name string,
	projectID uuid.UUID,
	providerID uuid.UUID,
	entityType pb.Entity,
) (*pb.EntityInstance, error) {
	// Convert pb.Entity to db.Entities
	dbEntityType, err := entities.EntityTypeToDBType(entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity type: %w", err)
	}

	// Get entity from database
	entity, err := s.store.GetEntityByName(ctx, db.GetEntityByNameParams{
		Name:       name,
		ProjectID:  projectID,
		ProviderID: providerID,
		EntityType: dbEntityType,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "entity not found")
		}
		return nil, fmt.Errorf("error fetching entity: %w", err)
	}

	// Get properties
	ewp, err := s.propSvc.EntityWithPropertiesByID(ctx, entity.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching properties for entity: %w", err)
	}

	if err := s.propSvc.RetrieveAllPropertiesForEntity(ctx, ewp, s.providerManager,
		propService.ReadBuilder().TolerateStaleData()); err != nil {
		return nil, fmt.Errorf("error fetching properties for entity: %w", err)
	}

	// Convert to protobuf
	return entityInstanceToProto(ewp), nil
}

func (s *entityService) DeleteEntityByID(
	ctx context.Context,
	entityID uuid.UUID,
	projectID uuid.UUID,
) error {
	// Get entity to verify it exists and belongs to the project
	entity, err := s.store.GetEntityByID(ctx, entityID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Errorf(codes.NotFound, "entity not found")
		}
		return fmt.Errorf("error fetching entity: %w", err)
	}

	// Verify entity belongs to the project
	if entity.ProjectID != projectID {
		return status.Errorf(codes.NotFound, "entity not found in project")
	}

	// Delete entity and its properties in a transaction
	tx, err := s.store.BeginTransaction()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error rolling back transaction")
		}
	}()

	qtx := s.store.GetQuerierWithTransaction(tx)

	// Delete properties first
	if err := qtx.DeleteAllPropertiesForEntity(ctx, entityID); err != nil {
		return fmt.Errorf("error deleting entity properties: %w", err)
	}

	// Delete entity
	if err := qtx.DeleteEntity(ctx, db.DeleteEntityParams{
		ID:        entityID,
		ProjectID: projectID,
	}); err != nil {
		return fmt.Errorf("error deleting entity: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// Helper functions

// entityInstanceToProto converts an EntityWithProperties to a pb.EntityInstance
func entityInstanceToProto(ewp *models.EntityWithProperties) *pb.EntityInstance {
	// Convert properties to structpb.Struct
	propsStruct := ewp.Properties.ToProtoStruct()

	return &pb.EntityInstance{
		Id: ewp.Entity.ID.String(),
		Context: &pb.ContextV2{
			ProjectId: ewp.Entity.ProjectID.String(),
			Provider:  "", // This would need to be filled in from the provider name
		},
		Name:       ewp.Entity.Name,
		Type:       ewp.Entity.Type,
		Properties: propsStruct,
	}
}
