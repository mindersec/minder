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
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/verifier"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	// ArtifactTypeContainerRetentionPeriod represents the retention period for container artifacts
	ArtifactTypeContainerRetentionPeriod = time.Now().AddDate(0, -6, 0)
)

// RepoReconcilerEvent is an event that is sent to the reconciler topic
type RepoReconcilerEvent struct {
	// Project is the project that the event is relevant to
	Project uuid.UUID `json:"project"`
	// Repository is the repository to be reconciled
	Repository int32 `json:"repository" validate:"gte=0"`
}

// NewRepoReconcilerMessage creates a new repos init event
func NewRepoReconcilerMessage(provider string, repoID int32, projectID uuid.UUID) (*message.Message, error) {
	evt := &RepoReconcilerEvent{
		Repository: repoID,
		Project:    projectID,
	}

	evtStr, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling init event: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), evtStr)
	msg.Metadata.Set("provider", provider)
	return msg, nil
}

// handleRepoReconcilerEvent handles events coming from the reconciler topic
func (e *Reconciler) handleRepoReconcilerEvent(msg *message.Message) error {
	var evt RepoReconcilerEvent
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
	return e.handleArtifactsReconcilerEvent(ctx, &evt)
}

// HandleArtifactsReconcilerEvent recreates the artifacts belonging to
// an specific repository
// nolint: gocyclo
func (e *Reconciler) handleArtifactsReconcilerEvent(ctx context.Context, evt *RepoReconcilerEvent) error {
	// first retrieve data for the repository
	repository, err := e.store.GetRepositoryByRepoID(ctx, evt.Repository)
	if err != nil {
		return fmt.Errorf("error retrieving repository: %w", err)
	}

	prov, err := e.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      repository.Provider,
		ProjectID: evt.Project,
	})
	if err != nil {
		return fmt.Errorf("error retrieving provider: %w", err)
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(e.provMt),
	}
	p, err := providers.GetProviderBuilder(ctx, prov, evt.Project, e.store, e.crypteng, pbOpts...)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	// evaluate profile for repo
	repo := util.PBRepositoryFromDB(repository)

	err = entities.NewEntityInfoWrapper().
		WithProvider(prov.Name).
		WithRepository(repo).
		WithProjectID(evt.Project).
		WithRepositoryID(repository.ID).
		Publish(e.evt)
	if err != nil {
		return fmt.Errorf("error publishing message: %w", err)
	}

	if !p.Implements(db.ProviderTypeGithub) {
		log.Printf("provider %s is not supported for artifacts reconciler", prov.Name)
		return nil
	}

	cli, err := p.GetGitHub(ctx)
	if err != nil {
		return fmt.Errorf("error getting github client: %w", err)
	}

	isOrg := (cli.GetOwner() != "")
	// todo: add another type of artifacts
	artifacts, err := cli.ListPackagesByRepository(ctx, isOrg, repository.RepoOwner,
		string(verifier.ArtifactTypeContainer), int64(repository.RepoID), 1, 100)
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			// we do not return error since it's a valid use case for a repository to not have artifacts
			log.Printf("error retrieving artifacts for RepoID %d: %v", repository.RepoID, err)
			return nil
		}
		return err
	}

	for _, artifact := range artifacts {
		// store information if we do not have it
		newArtifact, err := e.store.UpsertArtifact(ctx,
			db.UpsertArtifactParams{RepositoryID: repository.ID, ArtifactName: artifact.GetName(),
				ArtifactType: artifact.GetPackageType(), ArtifactVisibility: artifact.GetVisibility()})

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
			WithProvider(prov.Name).
			WithArtifact(pbArtifact).
			WithProjectID(evt.Project).
			WithArtifactID(newArtifact.ID).
			WithRepositoryID(repository.ID).
			Publish(e.evt)
		if err != nil {
			return fmt.Errorf("error publishing message: %w", err)
		}
	}
	return nil
}
