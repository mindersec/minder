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

// Package artifacts stores logic relating to the artifact entity type
package artifacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util/ptr"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GetArtifact retrieves an artifact and its versions from the database
// Returns provider ID alongside the artifact
func GetArtifact(
	ctx context.Context,
	store db.Querier,
	projectID uuid.UUID,
	artifactID uuid.UUID,
) (uuid.UUID, *minderv1.Artifact, error) {
	// Retrieve artifact details
	artifact, err := store.GetArtifactByID(ctx, db.GetArtifactByIDParams{
		ID:        artifactID,
		ProjectID: projectID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil, fmt.Errorf("artifact not found")
	} else if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to get artifact: %v", err)
	}

	var repoOwner, repoName string
	if artifact.RepositoryID.Valid {
		dbrepo, err := store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
			ID:        artifact.RepositoryID.UUID,
			ProjectID: projectID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, nil, fmt.Errorf("repository not found")
		} else if err != nil {
			return uuid.Nil, nil, fmt.Errorf("cannot read repository: %v", err)
		}

		repoOwner = dbrepo.RepoOwner
		repoName = dbrepo.RepoName
	}

	// Build the artifact protobuf
	return artifact.ProviderID, &minderv1.Artifact{
		ArtifactPk: artifact.ID.String(),
		Context: &minderv1.Context{
			Project:  ptr.Ptr(projectID.String()),
			Provider: ptr.Ptr(artifact.ProviderName),
		},
		Owner:      repoOwner,
		Name:       artifact.ArtifactName,
		Type:       artifact.ArtifactType,
		Visibility: artifact.ArtifactVisibility,
		Repository: repoName,
		CreatedAt:  timestamppb.New(artifact.CreatedAt),
	}, nil
}
