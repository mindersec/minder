// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/entities/properties"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/providers"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/util"
)

// ReconcileEntityRegistration reconciles the registration of an entity.
//
// Currently, this method only supports repositories but is intended to be
// generic and handle all types of entities.
// Todo: Utilise for other entities when such are supported.
func (s *Server) ReconcileEntityRegistration(
	ctx context.Context,
	in *pb.ReconcileEntityRegistrationRequest,
) (*pb.ReconcileEntityRegistrationResponse, error) {
	l := zerolog.Ctx(ctx).With().Logger()

	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	logger.BusinessRecord(ctx).Project = projectID

	// Todo: We don't support other entities yet. This should be updated when we do.
	entityType := in.GetEntity()
	if pb.EntityFromString(entityType) != pb.Entity_ENTITY_REPOSITORIES {
		return nil, util.UserVisibleError(codes.InvalidArgument, "entity type %s not supported", entityType)
	}

	providerNameParam := in.GetContext().GetProvider()
	provs, errorProvs, err := s.providerManager.BulkInstantiateByTrait(
		ctx, projectID, db.ProviderTypeRepoLister, providerNameParam)
	if err != nil {
		pErr := providers.ErrProviderNotFoundBy{}
		if errors.As(err, &pErr) {
			return nil, util.UserVisibleError(codes.NotFound, "no suitable provider found, please enroll a provider")
		}
		return nil, providerError(err)
	}

	for providerID, providerT := range provs {
		repos, err := s.fetchRepositoriesForProvider(ctx, projectID, providerID, providerT.Name, providerT.Provider)
		if err != nil {
			l.Error().
				Str("providerName", providerT.Name).
				Str("projectID", projectID.String()).
				Err(err).
				Msg("error fetching repositories for provider")
			errorProvs = append(errorProvs, providerT.Name)
			continue
		}

		for _, repo := range repos {
			if repo.Repo.Registered {
				continue
			}

			msg, err := createEntityMessage(ctx, &l, projectID, providerID, repo.Entity.GetEntity().GetProperties())
			if err != nil {
				l.Error().Err(err).
					Int64("repoID", repo.Repo.RepoId).
					Str("providerName", providerT.Name).
					Msg("error creating registration entity message")
				// This message will not be sent, but we can continue with the rest.
				continue
			}

			if err := s.publishEntityMessage(&l, msg); err != nil {
				l.Error().Err(err).Str("messageID", msg.UUID).Msg("error publishing register entities message")
			}
		}
	}

	// If all providers failed, return an error
	if len(errorProvs) > 0 && len(provs) == len(errorProvs) {
		return nil, util.UserVisibleError(codes.Internal, "cannot register entities for providers: %v", errorProvs)
	}

	return &pb.ReconcileEntityRegistrationResponse{}, nil
}

func (s *Server) publishEntityMessage(l *zerolog.Logger, msg *message.Message) error {
	l.Info().Str("messageID", msg.UUID).Msg("publishing register entities message for execution")
	return s.evt.Publish(events.TopicQueueReconcileEntityAdd, msg)
}

func createEntityMessage(
	ctx context.Context,
	l *zerolog.Logger,
	projectID, providerID uuid.UUID,
	props *structpb.Struct,
) (*message.Message, error) {
	msg := message.NewMessage(uuid.New().String(), nil)
	msg.SetContext(ctx)

	repoProps, err := properties.NewProperties(props.AsMap())
	if err != nil {
		return nil, err
	}

	event := messages.NewMinderEvent().
		WithProjectID(projectID).
		WithProviderID(providerID).
		WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
		WithProperties(repoProps)

	err = event.ToMessage(msg)
	if err != nil {
		l.Error().Err(err).Msg("error marshalling register entities message")
		return nil, err
	}

	return msg, nil
}
