// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"fmt"

	"github.com/mindersec/minder/internal/db"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// EntityTypeToDBType converts a pb.Entity to a db.Entities
// Returns an error if the entity type is not recognized or if it's ENTITY_UNSPECIFIED
func EntityTypeToDBType(entityType pb.Entity) (db.Entities, error) {
	switch entityType {
	case pb.Entity_ENTITY_REPOSITORIES:
		return db.EntitiesRepository, nil
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return db.EntitiesBuildEnvironment, nil
	case pb.Entity_ENTITY_ARTIFACTS:
		return db.EntitiesArtifact, nil
	case pb.Entity_ENTITY_PULL_REQUESTS:
		return db.EntitiesPullRequest, nil
	case pb.Entity_ENTITY_RELEASE:
		return db.EntitiesRelease, nil
	case pb.Entity_ENTITY_PIPELINE_RUN:
		return db.EntitiesPipelineRun, nil
	case pb.Entity_ENTITY_TASK_RUN:
		return db.EntitiesTaskRun, nil
	case pb.Entity_ENTITY_BUILD:
		return db.EntitiesBuild, nil
	case pb.Entity_ENTITY_UNSPECIFIED:
		return db.Entities(""), fmt.Errorf("invalid entity type: ENTITY_UNSPECIFIED is not a valid entity type")
	default:
		return db.Entities(""), fmt.Errorf("invalid entity type: %s is not a recognized entity type", entityType.String())
	}
}
