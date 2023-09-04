// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/stacklok/mediator/pkg/container"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/providers"
)

// CONTAINER_TYPE is the type for container artifacts
var CONTAINER_TYPE = "container"

// HandleArtifactsReconcilerEvent recreates the artifacts belonging to
// an specific repository
// nolint: gocyclo
func (e *Executor) HandleArtifactsReconcilerEvent(ctx context.Context, prov string, evt *ReconcilerEvent) error {
	cli, err := providers.BuildClient(ctx, prov, evt.Group, e.querier)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	// first retrieve data for the repository
	repository, err := e.querier.GetRepositoryByRepoID(ctx, db.GetRepositoryByRepoIDParams{
		Provider: prov,
		RepoID:   evt.Repository,
	})
	if err != nil {
		return fmt.Errorf("error retrieving repository: %w", err)
	}
	isOrg := (cli.GetOwner() != "")
	// todo: add another type of artifacts
	artifacts, err := cli.ListPackagesByRepository(ctx, isOrg, repository.RepoOwner,
		CONTAINER_TYPE, int64(repository.RepoID), 1, 100)
	if err != nil {
		return fmt.Errorf("error retrieving artifacts: %w", err)
	}
	for _, artifact := range artifacts.Packages {
		// store information if we do not have it
		newArtifact, err := e.querier.UpsertArtifact(ctx,
			db.UpsertArtifactParams{RepositoryID: repository.ID, ArtifactName: artifact.GetName(),
				ArtifactType: artifact.GetPackageType(), ArtifactVisibility: artifact.GetVisibility()})

		if err != nil {
			// just log error and continue
			log.Printf("error storing artifact: %v", err)
			continue
		}

		// remove older versions
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		err = e.querier.DeleteOldArtifactVersions(ctx,
			db.DeleteOldArtifactVersionsParams{ArtifactID: int32(artifact.GetID()), CreatedAt: thirtyDaysAgo})
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
			if container.TagIsSignature(tags) {
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

			_, err = e.querier.UpsertArtifactVersion(ctx, db.UpsertArtifactVersionParams{ArtifactID: newArtifact.ID, Version: *version.ID,
				Tags: sql.NullString{Valid: true, String: tagNames}, Sha: *version.Name, SignatureVerification: sigInfo,
				GithubWorkflow: workflowInfo, CreatedAt: version.CreatedAt.Time})
			if err != nil {
				// just log error and continue
				log.Printf("error storing artifact version: %v", err)
				continue
			}
		}
	}
	return nil
}
