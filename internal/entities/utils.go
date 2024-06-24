// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package entities contains logic relating to entity management
package entities

import (
	"errors"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
)

// EntityFromIDs takes the IDs of the three known entity types and
// returns a single ID along with the type of the entity.
// This assumes that exactly one of the IDs is not equal to uuid.Nil
func EntityFromIDs(
	repositoryID uuid.UUID,
	artifactID uuid.UUID,
	pullRequestID uuid.UUID,
) (uuid.UUID, db.Entities, error) {
	// Note that the repo ID is often passed around with PRs.
	// As a result, we test PRs and artifacts first, then repos.
	if pullRequestID != uuid.Nil {
		return pullRequestID, db.EntitiesPullRequest, nil
	}
	if artifactID != uuid.Nil {
		return artifactID, db.EntitiesArtifact, nil
	}
	if repositoryID != uuid.Nil {
		return repositoryID, db.EntitiesRepository, nil
	}
	return uuid.Nil, "", errors.New("all entity IDs are nil")
}
