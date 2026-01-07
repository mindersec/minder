// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/entities/models"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// ListEntities returns a list of entity instances for a given project and provider
func (s *Server) ListEntities(
	ctx context.Context,
	in *pb.ListEntitiesRequest,
) (*pb.ListEntitiesResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	logger.BusinessRecord(ctx).Provider = providerName
	logger.BusinessRecord(ctx).Project = projectID

	// Get provider ID from name
	provider, err := s.providerStore.GetByName(ctx, projectID, providerName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "provider not found")
		}
		return nil, fmt.Errorf("error getting provider: %w", err)
	}

	// Get entity type from request
	entityType := in.GetEntityType()
	if entityType == pb.Entity_ENTITY_UNSPECIFIED {
		return nil, util.UserVisibleError(codes.InvalidArgument, "entity type must be specified")
	}

	// Get limit from request
	limit := in.GetCursor().GetSize()
	if limit == 0 {
		limit = 20 // Default limit
	}

	// Get cursor from request
	cursor := in.GetCursor().GetCursor()

	// Call service to get entities
	outentities, nextCursor, err := s.entityService.ListEntities(
		ctx,
		projectID,
		provider.ID,
		entityType,
		cursor,
		int64(limit),
	)
	if err != nil {
		return nil, err
	}

	// Create response
	resp := &pb.ListEntitiesResponse{
		Results: outentities,
		Page: &pb.CursorPage{
			Next: &pb.Cursor{
				Cursor: nextCursor,
				Size:   limit,
			},
		},
	}

	return resp, nil
}

// GetEntityById returns an entity instance for a given entity ID
func (s *Server) GetEntityById(
	ctx context.Context,
	in *pb.GetEntityByIdRequest,
) (*pb.GetEntityByIdResponse, error) {
	// Parse entity ID
	entityID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid entity ID")
	}

	projectID := GetProjectID(ctx)

	// Call service to get entity
	entity, err := s.entityService.GetEntityByID(ctx, entityID, projectID)
	if err != nil {
		return nil, err
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID
	logger.BusinessRecord(ctx).Entity = entityID

	return &pb.GetEntityByIdResponse{
		Entity: entity,
	}, nil
}

// GetEntityByName returns an entity instance for a given entity name
func (s *Server) GetEntityByName(
	ctx context.Context,
	in *pb.GetEntityByNameRequest,
) (*pb.GetEntityByNameResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	// Get provider ID from name
	provider, err := s.providerStore.GetByName(ctx, projectID, providerName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "provider not found")
		}
		return nil, fmt.Errorf("error getting provider: %w", err)
	}

	// Get entity type from request
	entityType := in.GetEntityType()
	if entityType == pb.Entity_ENTITY_UNSPECIFIED {
		return nil, util.UserVisibleError(codes.InvalidArgument, "entity type must be specified")
	}

	// Call service to get entity
	entity, err := s.entityService.GetEntityByName(
		ctx,
		in.GetName(),
		projectID,
		provider.ID,
		entityType,
	)
	if err != nil {
		return nil, err
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = providerName
	logger.BusinessRecord(ctx).Project = projectID

	return &pb.GetEntityByNameResponse{
		Entity: entity,
	}, nil
}

// DeleteEntityById deletes an entity instance for a given entity ID
func (s *Server) DeleteEntityById(
	ctx context.Context,
	in *pb.DeleteEntityByIdRequest,
) (*pb.DeleteEntityByIdResponse, error) {
	// Parse entity ID
	entityID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid entity ID")
	}

	projectID := GetProjectID(ctx)

	// Call service to delete entity
	err = s.entityService.DeleteEntityByID(ctx, entityID, projectID)
	if err != nil {
		return nil, err
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID
	logger.BusinessRecord(ctx).Entity = entityID

	return &pb.DeleteEntityByIdResponse{
		Id: in.GetId(),
	}, nil
}

// RegisterEntity creates a new entity instance
func (s *Server) RegisterEntity(
	ctx context.Context,
	in *pb.RegisterEntityRequest,
) (*pb.RegisterEntityResponse, error) {
	// 1. Extract context information
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	logger.BusinessRecord(ctx).Provider = providerName
	logger.BusinessRecord(ctx).Project = projectID

	// 2. Validate entity type
	if in.GetEntityType() == pb.Entity_ENTITY_UNSPECIFIED {
		return nil, util.UserVisibleError(codes.InvalidArgument,
			"entity_type must be specified")
	}

	// 3. Parse identifying properties
	identifyingProps, err := parseIdentifyingProperties(in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument,
			"invalid identifying_properties: %v", err)
	}

	// 4. Get provider from database
	provider, err := s.providerStore.GetByName(ctx, projectID, providerName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "provider not found")
		}
		return nil, util.UserVisibleError(codes.Internal, "cannot get provider: %v", err)
	}

	// 5. Create entity using EntityCreator service
	ewp, err := s.entityCreator.CreateEntity(ctx, provider, projectID,
		in.GetEntityType(), identifyingProps, nil) // Use default options
	if err != nil {
		// If the error is already a UserVisibleError, pass it through directly.
		// This allows providers and EntityCreator to add user-visible errors
		// without needing to update this allow-list.
		var userErr *util.NiceStatus
		if errors.As(err, &userErr) {
			return nil, err
		}
		return nil, util.UserVisibleError(codes.Internal,
			"unable to register entity: %v", err)
	}

	// 6. Convert to EntityInstance protobuf
	entityInstance := entityInstanceToProto(ewp, providerName)

	// 7. Return response
	return &pb.RegisterEntityResponse{
		Entity: entityInstance,
	}, nil
}

// parseIdentifyingProperties converts proto properties to Properties object
func parseIdentifyingProperties(req *pb.RegisterEntityRequest) (*properties.Properties, error) {
	identifyingProps := req.GetIdentifyingProperties()
	if identifyingProps == nil {
		return nil, errors.New("identifying_properties is required")
	}

	// Validate total size to prevent resource exhaustion
	// Using proto.Size provides a better bound than counting properties,
	// as it accounts for arbitrarily large values in a Struct
	const maxProtoSize = 32 * 1024 // 32KB should be plenty for identifying properties
	if protoSize := proto.Size(identifyingProps); protoSize > maxProtoSize {
		return nil, fmt.Errorf("identifying_properties too large: %d bytes, max %d bytes",
			protoSize, maxProtoSize)
	}

	propsMap := identifyingProps.AsMap()

	// Validate property keys are reasonable length
	for key := range propsMap {
		if len(key) > 200 {
			return nil, fmt.Errorf("property key too long: %d characters", len(key))
		}
	}

	return properties.NewProperties(propsMap), nil
}

// entityInstanceToProto converts EntityWithProperties to EntityInstance protobuf
func entityInstanceToProto(ewp *models.EntityWithProperties, providerName string) *pb.EntityInstance {
	return &pb.EntityInstance{
		Id: ewp.Entity.ID.String(),
		Context: &pb.ContextV2{
			ProjectId: ewp.Entity.ProjectID.String(),
			Provider:  providerName,
		},
		Type: ewp.Entity.Type,
		Name: ewp.Entity.Name,
		// Properties are intentionally omitted - use GetEntityById to fetch them
	}
}
