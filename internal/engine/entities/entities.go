// Copyright 2023 Stacklok, Inc.
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
// Package rule provides the CLI subcommand for managing rules

// Package entities contains internal helper functions to deal with,
// validate and print the Entity protobuf enum. Mostly to interact
// with the database.
package entities

import (
	"strings"

	"golang.org/x/exp/slices"

	"github.com/stacklok/minder/internal/db"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		// This shouldn't happen
	}

	return dbEnt
}
