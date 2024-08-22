// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/engcontext"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/reconcilers/messages"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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

	providerName := in.GetContext().GetProvider()
	provs, errorProvs, err := s.providerManager.BulkInstantiateByTrait(ctx, projectID, db.ProviderTypeRepoLister, providerName)
	if err != nil {
		pErr := providers.ErrProviderNotFoundBy{}
		if errors.As(err, &pErr) {
			return nil, util.UserVisibleError(codes.NotFound, "no suitable provider found, please enroll a provider")
		}
		return nil, providerError(err)
	}

	for providerName, provider := range provs {
		// Explicitly fetch the provider here as we need its ID for posting the event.
		pvr, err := s.providerStore.GetByName(ctx, projectID, providerName)
		if err != nil {
			errorProvs = append(errorProvs, providerName)
			continue
		}

		repos, err := s.fetchRepositoriesForProvider(ctx, projectID, providerName, provider)
		if err != nil {
			l.Error().
				Str("providerName", providerName).
				Str("projectID", projectID.String()).
				Err(err).
				Msg("error fetching repositories for provider")
			errorProvs = append(errorProvs, providerName)
			continue
		}

		for _, repo := range repos {
			if repo.Registered {
				continue
			}

			msg, err := createEntityMessage(ctx, &l, projectID, pvr.ID, repo.GetName(), repo.GetOwner())
			if err != nil {
				l.Error().Err(err).
					Int64("repoID", repo.RepoId).
					Str("providerName", providerName).
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
	repoName, repoOwner string,
) (*message.Message, error) {
	msg := message.NewMessage(uuid.New().String(), nil)
	msg.SetContext(ctx)

	event := messages.NewMinderEvent[*messages.RepoEvent]().
		WithProjectID(projectID).
		WithProviderID(providerID).
		WithEntity(messages.NewRepoEvent().
			WithRepoName(repoName).
			WithRepoOwner(repoOwner),
		)

	err := event.ToMessage(msg)
	if err != nil {
		l.Error().Err(err).Msg("error marshalling register entities message")
		return nil, err
	}

	return msg, nil
}
