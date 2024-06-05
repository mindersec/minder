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
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/reconcilers/messages"
	"github.com/stacklok/minder/internal/repositories"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// handleRepoReconcilerEvent handles events coming from the reconciler topic
func (r *Reconciler) handleRepoReconcilerEvent(msg *message.Message) error {
	var evt messages.RepoReconcilerEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
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
	log.Printf("handling reconciler event for project %s and repository %d", evt.Project.String(), evt.Repository)
	return r.handleArtifactsReconcilerEvent(ctx, &evt)
}

// HandleArtifactsReconcilerEvent recreates the artifacts belonging to
// an specific repository
// nolint: gocyclo
func (r *Reconciler) handleArtifactsReconcilerEvent(ctx context.Context, evt *messages.RepoReconcilerEvent) error {
	// first retrieve data for the repository
	repository, err := r.store.GetRepositoryByRepoID(ctx, evt.Repository)
	if err != nil {
		return fmt.Errorf("error retrieving repository: %w", err)
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

	// evaluate profile for repo
	repo := repositories.PBRepositoryFromDB(repository)

	err = entities.NewEntityInfoWrapper().
		WithProviderID(providerID).
		WithRepository(repo).
		WithProjectID(evt.Project).
		WithRepositoryID(repository.ID).
		Publish(r.evt)
	if err != nil {
		return fmt.Errorf("error publishing message: %w", err)
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
		newArtifact, err := r.store.UpsertArtifact(ctx,
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
			// just log error and continue
			log.Printf("error storing artifact: %v", err)
			continue
		}

		// publish event for artifact
		pbArtifact := &pb.Artifact{
			ArtifactPk: newArtifact.ID.String(),
			Owner:      *artifact.GetOwner().Login,
			Name:       artifact.GetName(),
			Type:       artifact.GetPackageType(),
			Visibility: artifact.GetVisibility(),
			Repository: repository.RepoName,
			Versions:   nil, // explicitly nil, will be filled by the ingester
			CreatedAt:  timestamppb.New(artifact.GetCreatedAt().Time),
		}
		err = entities.NewEntityInfoWrapper().
			WithProviderID(providerID).
			WithArtifact(pbArtifact).
			WithProjectID(evt.Project).
			WithArtifactID(newArtifact.ID).
			WithRepositoryID(repository.ID).
			Publish(r.evt)
		if err != nil {
			return fmt.Errorf("error publishing message: %w", err)
		}
	}
	return nil
}
