// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/logger"
	reconcilers "github.com/mindersec/minder/internal/reconcilers/messages"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

// CreateEntityReconciliationTask creates a task to reconcile the state of an entity
func (s *Server) CreateEntityReconciliationTask(ctx context.Context,
	in *pb.CreateEntityReconciliationTaskRequest) (
	*pb.CreateEntityReconciliationTaskResponse, error,
) {
	// Populated by EntityContextProjectInterceptor using incoming request
	entityCtx := engcontext.EntityFromContext(ctx)
	dbProvider, err := entityCtx.Validate(ctx, s.store, s.providerStore)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	entity := in.GetEntity()

	if entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}

	var entityType db.Entities
	var topic string

	switch entity.GetType() {
	case pb.Entity_ENTITY_REPOSITORIES:
		entityType = db.EntitiesRepository
		topic = constants.TopicQueueReconcileRepoInit
	case pb.Entity_ENTITY_ARTIFACTS:
		entityType = db.EntitiesArtifact
		topic = constants.TopicQueueReconcileEntityAdd
	case pb.Entity_ENTITY_PULL_REQUESTS:
		entityType = db.EntitiesPullRequest
		topic = constants.TopicQueueReconcileEntityAdd
	case pb.Entity_ENTITY_UNSPECIFIED,
		pb.Entity_ENTITY_BUILD_ENVIRONMENTS,
		pb.Entity_ENTITY_RELEASE,
		pb.Entity_ENTITY_PIPELINE_RUN,
		pb.Entity_ENTITY_TASK_RUN,
		pb.Entity_ENTITY_BUILD:
		return nil, status.Errorf(codes.InvalidArgument, "entity type %s is not supported", entity.GetType())
	default:
		return nil, status.Errorf(codes.InvalidArgument, "entity type %s is not supported", entity.GetType())
	}

	if entity.GetId() == "" {
		// Look up ID given name
		dbEntity, err := s.store.GetEntityByName(ctx, db.GetEntityByNameParams{
			ProjectID:  entityCtx.Project.ID,
			EntityType: entityType,
			Name:       entity.GetName(),
			ProviderID: dbProvider.ID,
		})
		if err != nil {
			return nil, util.UserVisibleError(codes.NotFound,
				"Unable to find entity %q of type %s in provider %s",
				entity.GetName(),
				entityType,
				entityCtx.Provider.Name,
			)
		}
		entity.Id = dbEntity.ID.String()
	}

	msg, err := getEntityReconciliationMessage(ctx, s.store, entity.GetId(), entityCtx)
	if err != nil {
		return nil, err
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := s.evt.Publish(topic, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}

	return &pb.CreateEntityReconciliationTaskResponse{}, nil
}

func getEntityReconciliationMessage(ctx context.Context, store db.Store,
	repoIdString string, entityCtx engcontext.EntityContext) (*message.Message, error) {
	repoUUID, err := uuid.Parse(repoIdString)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error parsing entity id: %v", err)
	}

	// Fetch entity by ID
	ent, err := store.GetEntityByID(ctx, repoUUID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "entity not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot read entity: %v", err)
	}

	// Verify project matches
	if ent.ProjectID != entityCtx.Project.ID {
		return nil, status.Errorf(codes.NotFound, "entity not found")
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = ent.ProviderID
	logger.BusinessRecord(ctx).Project = ent.ProjectID

	if ent.EntityType == db.EntitiesRepository {
		logger.BusinessRecord(ctx).Repository = ent.ID
		msg, err := reconcilers.NewRepoReconcilerMessage(ent.ProviderID, ent.ID, ent.ProjectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting reconciler message: %v", err)
		}
		return msg, nil
	}

	// For non-repository entities, use MinderEvent
	minderEvt := reconcilers.NewMinderEvent().
		WithProviderID(ent.ProviderID).
		WithProjectID(ent.ProjectID).
		WithEntityID(ent.ID)

	msg := message.NewMessage(uuid.New().String(), nil)
	if err := minderEvt.ToMessage(msg); err != nil {
		return nil, status.Errorf(codes.Internal, "error getting reconciler message: %v", err)
	}
	return msg, nil
}
