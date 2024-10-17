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
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/logger"
	reconcilers "github.com/mindersec/minder/internal/reconcilers/messages"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// CreateEntityReconciliationTask creates a task to reconcile the state of an entity
func (s *Server) CreateEntityReconciliationTask(ctx context.Context,
	in *pb.CreateEntityReconciliationTaskRequest) (
	*pb.CreateEntityReconciliationTaskResponse, error,
) {
	// Populated by EntityContextProjectInterceptor using incoming request
	entityCtx := engcontext.EntityFromContext(ctx)
	err := entityCtx.Validate(ctx, s.store, s.providerStore)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	entity := in.GetEntity()

	if entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}

	var msg *message.Message
	var topic string

	// TODO: Support other entity types, replace with switch
	if entity.GetType() == pb.Entity_ENTITY_REPOSITORIES {
		msg, err = getRepositoryReconciliationMessage(ctx, s.store, entity.GetId(), entityCtx)
		if err != nil {
			return nil, err
		}
		topic = events.TopicQueueReconcileRepoInit
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "entity type %s is not supported", entity.GetType())
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := s.evt.Publish(topic, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}

	return &pb.CreateEntityReconciliationTaskResponse{}, nil
}

func getRepositoryReconciliationMessage(ctx context.Context, store db.Store,
	repoIdString string, entityCtx engcontext.EntityContext) (*message.Message, error) {
	repoUUID, err := uuid.Parse(repoIdString)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error parsing repository id: %v", err)
	}

	repo, err := store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
		ID:        repoUUID,
		ProjectID: entityCtx.Project.ID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot read repository: %v", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = repo.ProviderID
	logger.BusinessRecord(ctx).Project = repo.ProjectID
	logger.BusinessRecord(ctx).Repository = repo.ID

	msg, err := reconcilers.NewRepoReconcilerMessage(repo.ProviderID, repo.ID, repo.ProjectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting reconciler message: %v", err)
	}

	return msg, nil
}
