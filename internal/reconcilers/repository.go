// Copyright 2023 Stacklok, Inc
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

package reconcilers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	entityMessage "github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/reconcilers/messages"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// handleRepoReconcilerEvent handles events coming from the reconciler topic
func (r *Reconciler) handleRepoReconcilerEvent(msg *message.Message) error {
	var evt messages.RepoReconcilerEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		log.Printf("error unmarshalling event: %v", err)
		return nil
	}

	// validate event
	validate := validator.New()
	if err := validate.Struct(&evt); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		log.Printf("error validating event: %v", err)
		return nil
	}

	ctx := msg.Context()
	log.Printf("handling reconciler event for project %s and repository %s", evt.Project.String(), evt.EntityID.String())
	return r.handleArtifactsReconcilerEvent(ctx, &evt)
}

// HandleArtifactsReconcilerEvent recreates the artifacts belonging to
// an specific repository
// nolint: gocyclo
func (r *Reconciler) handleArtifactsReconcilerEvent(ctx context.Context, evt *messages.RepoReconcilerEvent) error {
	entRefresh := entityMessage.NewEntityRefreshAndDoMessage().
		WithEntityID(evt.EntityID)

	m := message.NewMessage(uuid.New().String(), nil)
	if err := entRefresh.ToMessage(m); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error marshalling message")
		// no point in retrying, so we return nil
		return nil
	}

	if evt.EntityID == uuid.Nil {
		// this might happen if we process old messages during an upgrade, but there's no point in retrying
		zerolog.Ctx(ctx).Error().Msg("entityID is nil")
		return nil
	}

	m.SetContext(ctx)
	if err := r.evt.Publish(events.TopicQueueRefreshEntityByIDAndEvaluate, m); err != nil {
		// we retry in case watermill is having a bad day
		return fmt.Errorf("error publishing message: %w", err)
	}

	// the code below this line will be refactored in a follow-up PR
	ewp, err := r.propService.EntityWithPropertiesByID(ctx, evt.EntityID, nil)
	if err != nil {
		// database might be down, so we retry
		log.Printf("error retrieving entity with properties: %v", err)
		return nil
	}

	repoID, err := ewp.Properties.GetProperty(properties.PropertyUpstreamID).AsInt64()
	if err != nil {
		log.Printf("error getting property upstreamID: %v", err)
		// no point in retrying, so we just return nil
		return nil
	}

	// first retrieve data for the repository
	repository, err := r.store.GetRepositoryByRepoID(ctx, repoID)
	if errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Debug().Err(err).
			Int64("repositoryUpstreamID", repoID).
			Msg("repository not found")
		return nil
	}
	if err != nil {
		// database might be down, so we retry
		return fmt.Errorf("error retrieving repository %d in project %s: %w", repoID, evt.Project, err)
	}

	providerID := repository.ProviderID

	p, err := r.providerManager.InstantiateFromID(ctx, providerID)
	if err != nil {
		return fmt.Errorf("error instantiating provider: %w", err)
	}

	cli, err := v1.As[v1.GitHub](p)
	if err != nil {
		// Keeping this behaviour to match the existing logic
		// This reconciler logic needs to be split between GitHub-specific
		// and generic parts.
		log.Printf("provider %s is not supported for artifacts reconciler", providerID)
		return nil
	}

	// todo: add another type of artifacts
	artifacts, err := cli.ListPackagesByRepository(ctx, repository.RepoOwner, string(verifyif.ArtifactTypeContainer),
		int64(repository.RepoID), 1, 100)
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			// we do not return error since it's a valid use case for a repository to not have artifacts
			log.Printf("error retrieving artifacts for RepoID %d: %v", repository.RepoID, err)
			return nil
		} else if errors.Is(err, github.ErrNoPackageListingClient) {
			// not a hard error, just misconfiguration or the user doesn't want to put a token
			// into the provider config
			zerolog.Ctx(ctx).Info().
				Str("provider", providerID.String()).
				Msg("No package listing client available for provider")
			return nil
		}
		return err
	}

	for _, artifact := range artifacts {
		// store information if we do not have it
		typeLower := strings.ToLower(artifact.GetPackageType())
		var newArtifactID uuid.UUID
		pbArtifact, err := db.WithTransaction(r.store, func(tx db.ExtendQuerier) (*pb.Artifact, error) {
			newArtifact, err := tx.UpsertArtifact(ctx,
				db.UpsertArtifactParams{
					RepositoryID: uuid.NullUUID{
						UUID:  repository.ID,
						Valid: true,
					},
					ArtifactName:       artifact.GetName(),
					ArtifactType:       typeLower,
					ArtifactVisibility: artifact.GetVisibility(),
					ProjectID:          evt.Project,
					ProviderName:       repository.Provider,
					ProviderID:         providerID,
				})

			if err != nil {
				return nil, err
			}

			newArtifactID = newArtifact.ID

			// name is provider specific and should be based on properties.
			// In github's case it's lowercase owner / artifact name
			// TODO: Replace with a provider call to get
			// a name based on properties.
			var prefix string
			if artifact.GetOwner().GetLogin() != "" {
				prefix = artifact.GetOwner().GetLogin() + "/"
			}

			artName := prefix + artifact.GetName()

			_, err = tx.CreateOrEnsureEntityByID(ctx, db.CreateOrEnsureEntityByIDParams{
				ID:         newArtifact.ID,
				EntityType: db.EntitiesArtifact,
				Name:       artName,
				ProjectID:  evt.Project,
				ProviderID: providerID,
				OriginatedFrom: uuid.NullUUID{
					UUID:  repository.ID,
					Valid: true,
				},
			})
			if err != nil {
				return nil, err
			}

			// publish event for artifact
			return &pb.Artifact{
				ArtifactPk: newArtifact.ID.String(),
				Owner:      *artifact.GetOwner().Login,
				Name:       artifact.GetName(),
				Type:       artifact.GetPackageType(),
				Visibility: artifact.GetVisibility(),
				Repository: repository.RepoName,
				Versions:   nil, // explicitly nil, will be filled by the ingester
				CreatedAt:  timestamppb.New(artifact.GetCreatedAt().Time),
			}, nil
		})
		if err != nil {
			// just log error and continue
			log.Printf("error storing artifact: %v", err)
			continue
		}
		err = entities.NewEntityInfoWrapper().
			WithProviderID(providerID).
			WithArtifact(pbArtifact).
			WithProjectID(evt.Project).
			WithArtifactID(newArtifactID).
			WithRepositoryID(repository.ID).
			Publish(r.evt)
		if err != nil {
			return fmt.Errorf("error publishing message: %w", err)
		}
	}
	return nil
}
