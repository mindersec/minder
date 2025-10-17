// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/ptr"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// ListArtifacts lists all artifacts for a given project and provider
// nolint:gocyclo
func (s *Server) ListArtifacts(ctx context.Context, in *pb.ListArtifactsRequest) (*pb.ListArtifactsResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = providerName
	logger.BusinessRecord(ctx).Project = projectID

	artifactFilter, err := parseArtifactListFrom(s.store, in.From)
	if err != nil {
		return nil, fmt.Errorf("failed to parse artifact list from: %w", err)
	}

	results, err := artifactFilter.listArtifacts(ctx, providerName, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	return &pb.ListArtifactsResponse{Results: results}, nil
}

// GetArtifactByName gets an artifact by name
// nolint:gocyclo
func (s *Server) GetArtifactByName(ctx context.Context, in *pb.GetArtifactByNameRequest) (*pb.GetArtifactByNameResponse, error) {
	nameParts := strings.Split(in.Name, "/")
	if len(nameParts) < 3 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact name user repoOwner/repoName/artifactName")
	}

	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	logger.BusinessRecord(ctx).Provider = providerName

	// Get provider ID from name
	provider, err := s.providerStore.GetByName(ctx, projectID, providerName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "provider not found")
		}
		return nil, status.Errorf(codes.Internal, "cannot get provider: %v", err)
	}

	// the artifact name is the rest of the parts
	artifactName := strings.Join(nameParts[2:], "/")

	// Search for artifact by name property using V1 helper
	entities, err := s.store.GetTypedEntitiesByPropertyV1(
		ctx,
		db.EntitiesArtifact,
		properties.PropertyName,
		artifactName,
		db.GetTypedEntitiesOptions{
			ProjectID:  projectID,
			ProviderID: provider.ID,
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to search artifact: %s", err)
	}

	if len(entities) == 0 {
		return nil, status.Errorf(codes.NotFound, "artifact not found")
	}

	// Fetch the entity with properties
	ewp, err := s.props.EntityWithPropertiesByID(ctx, entities[0].ID, nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "artifact not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	// Retrieve all properties from provider
	if err := s.props.RetrieveAllPropertiesForEntity(ctx, ewp, s.providerManager, nil); err != nil {
		return nil, fmt.Errorf("error fetching properties for artifact: %w", err)
	}

	// Convert to protobuf
	somePB, err := s.props.EntityWithPropertiesAsProto(ctx, ewp, s.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	pbArtifact, ok := somePB.(*pb.Artifact)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", somePB)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = ewp.Entity.ProviderID
	logger.BusinessRecord(ctx).Project = ewp.Entity.ProjectID
	logger.BusinessRecord(ctx).Artifact = ewp.Entity.ID
	if ewp.Entity.OriginatedFrom != uuid.Nil {
		logger.BusinessRecord(ctx).Repository = ewp.Entity.OriginatedFrom
	}

	return &pb.GetArtifactByNameResponse{
		Artifact: pbArtifact,
		Versions: nil, // explicitly nil, will probably deprecate that field later
	}, nil
}

// GetArtifactById gets an artifact by id
// nolint:gocyclo
func (s *Server) GetArtifactById(ctx context.Context, in *pb.GetArtifactByIdRequest) (*pb.GetArtifactByIdResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	parsedArtifactID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact ID")
	}

	// Fetch artifact entity
	ewp, err := s.props.EntityWithPropertiesByID(ctx, parsedArtifactID, nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "artifact not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	// Verify the entity belongs to the correct project
	if ewp.Entity.ProjectID != projectID {
		return nil, status.Errorf(codes.NotFound, "artifact not found")
	}

	// Verify it's an artifact entity
	if ewp.Entity.Type != pb.Entity_ENTITY_ARTIFACTS {
		return nil, status.Errorf(codes.InvalidArgument, "entity is not an artifact")
	}

	// Retrieve all properties from provider
	if err := s.props.RetrieveAllPropertiesForEntity(ctx, ewp, s.providerManager, nil); err != nil {
		return nil, fmt.Errorf("error fetching properties for artifact: %w", err)
	}

	// Convert to protobuf
	somePB, err := s.props.EntityWithPropertiesAsProto(ctx, ewp, s.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	pbArtifact, ok := somePB.(*pb.Artifact)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", somePB)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = ewp.Entity.ProviderID
	logger.BusinessRecord(ctx).Project = ewp.Entity.ProjectID
	logger.BusinessRecord(ctx).Artifact = ewp.Entity.ID
	if ewp.Entity.OriginatedFrom != uuid.Nil {
		logger.BusinessRecord(ctx).Repository = ewp.Entity.OriginatedFrom
	}

	return &pb.GetArtifactByIdResponse{
		Artifact: pbArtifact,
		Versions: nil, // explicitly nil, will probably deprecate that field later
	}, nil
}

type artifactSource string

const (
	artifactSourceRepo artifactSource = "repository"
)

type artifactListFilter struct {
	store db.Store

	repoSlubList []string
	source       artifactSource
	filter       string
}

func parseArtifactListFrom(store db.Store, from string) (*artifactListFilter, error) {
	if from == "" {
		return &artifactListFilter{
			store:  store,
			source: artifactSourceRepo,
		}, nil
	}

	parts := strings.Split(from, "=")
	if len(parts) != 2 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid filter, use format: <source>=<filter>")
	}

	source := parts[0]
	filter := parts[1]

	var repoSlubList []string

	switch source {
	case string(artifactSourceRepo):
		repoSlubList = strings.Split(filter, ",")
	default:
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid filter source, only repository is supported")
	}

	return &artifactListFilter{
		store:        store,
		source:       artifactSource(source),
		filter:       filter,
		repoSlubList: repoSlubList,
	}, nil
}

func (filter *artifactListFilter) listArtifacts(
	ctx context.Context, providerName string, projectID uuid.UUID,
) ([]*pb.Artifact, error) {
	if filter.source != artifactSourceRepo {
		// just repos are supported now and we should never get here
		// when we support more, we turn this into an if-else or a switch
		return []*pb.Artifact{}, nil
	}

	// Get provider ID
	provider, err := filter.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:     providerName,
		Projects: []uuid.UUID{projectID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Get all artifact entities for this provider/project
	artifactEnts, err := filter.store.GetEntitiesByType(ctx, db.GetEntitiesByTypeParams{
		EntityType: db.EntitiesArtifact,
		ProviderID: provider.ID,
		Projects:   []uuid.UUID{projectID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact entities: %w", err)
	}

	// Filter by repository if needed and convert to protobuf
	results := []*pb.Artifact{}
	for _, ent := range artifactEnts {
		// Apply repository filter if specified
		if len(filter.repoSlubList) > 0 {
			// Check if artifact originates from one of the filtered repos
			if !ent.OriginatedFrom.Valid {
				continue
			}
			// We need to check if the originated_from repository matches the filter
			// For now, skip filtering - this requires loading the parent repo
			// TODO: Implement efficient repo filtering
		}

		// The entity name is the artifact name directly
		results = append(results, &pb.Artifact{
			ArtifactPk: ent.ID.String(),
			Context: &pb.Context{
				Provider: &providerName,
				Project:  ptr.Ptr(projectID.String()),
			},
			Owner:      "", // Will be populated by properties
			Name:       ent.Name,
			Type:       "", // Will be populated by properties
			Visibility: "", // Will be populated by properties
			Repository: "", // Will be populated by properties
			CreatedAt:  timestamppb.New(ent.CreatedAt),
		})
	}

	return results, nil
}
