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
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/container"
	"github.com/stacklok/mediator/pkg/db"
)

// HandleArtifactsReconcilerEvent recreates the artifacts belonging to
// an specific repository
// nolint: gocyclo
func (e *Executor) HandleArtifactsReconcilerEvent(ctx context.Context, prov string, evt *ReconcilerEvent) error {
	cli, token, owner_filter, err := e.buildClient(ctx, prov, evt.Group)
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
	isOrg := (owner_filter != "")
	// todo: add another type of artifacts
	artifacts, err := cli.ListPackagesByRepository(ctx, isOrg, repository.RepoOwner, "container", int64(repository.RepoID), 1, 100)
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
			if version.CreatedAt.After(thirtyDaysAgo) {
				tags := version.Metadata.Container.Tags
				// if the artifact has a .sig tag it's a signature, skip it
				found := false
				for _, tag := range tags {
					if strings.HasSuffix(tag, ".sig") {
						found = true
						break
					}
				}
				if found {
					continue
				}
				tagNames := strings.Join(tags, ",")

				// now get information for signature
				var sigInfo json.RawMessage
				var workflowInfo json.RawMessage

				imageRef := fmt.Sprintf("%s/%s/%s@%s", container.REGISTRY, *artifact.GetOwner().Login, artifact.GetName(), version.GetName())
				signature_verification, github_workflow, err := container.ValidateSignature(ctx,
					token, *artifact.GetOwner().Login, imageRef)
				if err != nil {
					// just log error and continue
					log.Printf("error validating signature: %v", err)
					continue
				}

				sig, err := util.GetBytesFromProto(signature_verification)
				if err != nil {
					// log error and just pass an empty json
					log.Printf("error getting bytes from proto: %v", err)
					sigInfo = json.RawMessage("{}")
				} else {
					sigInfo = json.RawMessage(sig)
				}
				work, err := util.GetBytesFromProto(github_workflow)
				if err != nil {
					// log error and just pass an empty json
					log.Printf("error getting bytes from proto: %v", err)
					workflowInfo = json.RawMessage("{}")
				} else {
					workflowInfo = json.RawMessage(work)
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
	}
	return nil

}
