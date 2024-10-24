// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package entities contains internal helper functions to deal with,
// validate and print the Entity protobuf enum. Mostly to interact
// with the database.
package entities

import (
	"strings"

	"golang.org/x/exp/slices"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/db"
)

// KnownTypesCSV returns a comma separated list of known entity types. Useful for UI
func KnownTypesCSV() string {
	var keys []string

	// Iterate through the map and append keys to the slice
	for _, pbval := range minderv1.Entity_value {
		ent := minderv1.Entity(pbval)
		// PRs are not a first-class object
		if !ent.IsValid() || ent == minderv1.Entity_ENTITY_PULL_REQUESTS {
			continue
		}
		keys = append(keys, ent.ToString())
	}

	slices.Sort(keys)
	return strings.Join(keys, ", ")
}

// EntityTypeFromDB returns the entity type from the database entity
func EntityTypeFromDB(entity db.Entities) minderv1.Entity {
	switch entity {
	case db.EntitiesRepository:
		return minderv1.Entity_ENTITY_REPOSITORIES
	case db.EntitiesBuildEnvironment:
		return minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS
	case db.EntitiesArtifact:
		return minderv1.Entity_ENTITY_ARTIFACTS
	case db.EntitiesPullRequest:
		return minderv1.Entity_ENTITY_PULL_REQUESTS
	case db.EntitiesRelease:
		return minderv1.Entity_ENTITY_RELEASE
	case db.EntitiesPipelineRun:
		return minderv1.Entity_ENTITY_PIPELINE_RUN
	case db.EntitiesTaskRun:
		return minderv1.Entity_ENTITY_TASK_RUN
	case db.EntitiesBuild:
		return minderv1.Entity_ENTITY_BUILD
	default:
		return minderv1.Entity_ENTITY_UNSPECIFIED
	}
}

// EntityTypeToDB returns the database entity from the protobuf entity type
func EntityTypeToDB(entity minderv1.Entity) db.Entities {
	var dbEnt db.Entities

	switch entity {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		dbEnt = db.EntitiesRepository
	case minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS:
		dbEnt = db.EntitiesBuildEnvironment
	case minderv1.Entity_ENTITY_ARTIFACTS:
		dbEnt = db.EntitiesArtifact
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		dbEnt = db.EntitiesPullRequest
	case minderv1.Entity_ENTITY_RELEASE:
		dbEnt = db.EntitiesRelease
	case minderv1.Entity_ENTITY_PIPELINE_RUN:
		dbEnt = db.EntitiesPipelineRun
	case minderv1.Entity_ENTITY_TASK_RUN:
		dbEnt = db.EntitiesTaskRun
	case minderv1.Entity_ENTITY_BUILD:
		dbEnt = db.EntitiesBuild
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		// This shouldn't happen
	}

	return dbEnt
}
