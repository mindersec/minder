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
	"log"
	"sort"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/container"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// CONTAINER_TYPE is the type for container artifacts
var CONTAINER_TYPE = "container"

// RepoReconcilerEvent is an event that is sent to the reconciler topic
type RepoReconcilerEvent struct {
	// Group is the group that the event is relevant to
	Group int32 `json:"group" validate:"gte=0"`
	// Repository is the repository to be reconciled
	Repository int32 `json:"repository" validate:"gte=0"`
}

// NewRepoReconcilerMessage creates a new repos init event
func NewRepoReconcilerMessage(provider string, repoID, groupID int32) (*message.Message, error) {
	evt := &RepoReconcilerEvent{
		Repository: repoID,
		Group:      groupID,
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
	log.Printf("handling reconciler event for group %d and repository %d", evt.Group, evt.Repository)
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
		Name:    repository.Provider,
		GroupID: evt.Group,
	})
	if err != nil {
		return fmt.Errorf("error retrieving provider: %w", err)
	}

	p, err := providers.GetProviderBuilder(ctx, prov, evt.Group, e.store, e.crypteng)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
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
		CONTAINER_TYPE, int64(repository.RepoID), 1, 100)
	if err != nil {
		return fmt.Errorf("error retrieving artifacts: %w", err)
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

		// remove older versions
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		err = e.store.DeleteOldArtifactVersions(ctx,
			db.DeleteOldArtifactVersionsParams{ArtifactID: newArtifact.ID, CreatedAt: thirtyDaysAgo})
		if err != nil {
			// just log error, we will not remove older for now
			log.Printf("error removing older artifact versions: %v", err)
		}

		// now query for versions, retrieve the ones from last month
		versions, err := cli.GetPackageVersions(ctx, isOrg, repository.RepoOwner, artifact.GetPackageType(), artifact.GetName())
		if err != nil {
			// just log error and continue
			log.Printf("error retrieving artifact versions: %v", err)
			continue
		}
		for _, version := range versions {
			if version.CreatedAt.Before(thirtyDaysAgo) {
				continue
			}

			tags := version.Metadata.Container.Tags
			if container.TagsContainSignature(tags) {
				continue
			}
			sort.Strings(tags)
			tagNames := strings.Join(tags, ",")

			// now get information for signature and workflow
			sigInfo, workflowInfo, err := container.GetArtifactSignatureAndWorkflowInfo(
				ctx, cli, *artifact.GetOwner().Login, artifact.GetName(), version.GetName())
			if errors.Is(err, container.ErrSigValidation) {
				// just log error and continue
				log.Printf("error validating signature: %v", err)
				continue
			} else if errors.Is(err, container.ErrProtoParse) {
				// log error and just pass an empty json
				log.Printf("error getting bytes from proto: %v", err)
			} else if err != nil {
				return fmt.Errorf("error getting signature and workflow info: %w", err)
			}

			newVersion, err := e.store.UpsertArtifactVersion(ctx,
				db.UpsertArtifactVersionParams{
					ArtifactID: newArtifact.ID,
					Version:    *version.ID,
					Tags:       sql.NullString{Valid: true, String: tagNames},
					Sha:        *version.Name, SignatureVerification: sigInfo,
					GithubWorkflow: workflowInfo,
					CreatedAt:      version.CreatedAt.Time,
				})
			if err != nil {
				// just log error and continue
				log.Printf("error storing artifact version: %v", err)
				continue
			}

			ghWorkflow := &pb.GithubWorkflow{}
			if err := protojson.Unmarshal(workflowInfo, ghWorkflow); err != nil {
				// just log error and continue
				log.Printf("error unmarshalling github workflow: %v", err)
				continue
			}

			sigVerification := &pb.SignatureVerification{}
			if err := protojson.Unmarshal(sigInfo, sigVerification); err != nil {
				log.Printf("error unmarshalling signature verification: %v", err)
				continue
			}

			versionedArtifact := &pb.VersionedArtifact{
				Artifact: &pb.Artifact{
					ArtifactPk: int64(newArtifact.ID),
					Owner:      *artifact.GetOwner().Login,
					Name:       artifact.GetName(),
					Type:       artifact.GetPackageType(),
					Visibility: artifact.GetVisibility(),
					Repository: repository.RepoName,
					CreatedAt:  timestamppb.New(artifact.GetCreatedAt().Time),
				},
				Version: &pb.ArtifactVersion{
					VersionId:             int64(newVersion.ID),
					Tags:                  tags,
					Sha:                   *version.Name,
					SignatureVerification: sigVerification,
					GithubWorkflow:        ghWorkflow,
					CreatedAt:             timestamppb.New(version.CreatedAt.Time),
				},
			}

			err = engine.NewEntityInfoWrapper().
				WithProvider(prov.Name).
				WithVersionedArtifact(versionedArtifact).
				WithGroupID(evt.Group).
				WithArtifactID(newArtifact.ID).
				WithRepositoryID(repository.ID).
				Publish(e.evt)
			if err != nil {
				return fmt.Errorf("error publishing message: %w", err)
			}
		}
	}
	return nil
}
