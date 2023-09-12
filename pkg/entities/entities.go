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
// Package rule provides the CLI subcommand for managing rules

// Package entities contains helper functions to convert to and from
// validate and print the Entity protobuf enum
package entities

import (
	"strings"

	"golang.org/x/exp/slices"

	"github.com/stacklok/mediator/internal/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// EntityType is the type of entity
type EntityType string

// Entity types as string-like enums. Used in CLI and other user-facing code
const (
	// RepositoryEntity is a repository entity
	RepositoryEntity EntityType = "repository"
	// BuildEnvironmentEntity is a build environment entity
	BuildEnvironmentEntity EntityType = "build_environment"
	// ArtifactEntity is an artifact entity
	ArtifactEntity EntityType = "artifact"
	// PullRequestEntity is a pull request entity
	PullRequestEntity EntityType = "pull_request"
	// UnknownEntity is an explicitly unknown entity
	UnknownEntity EntityType = "unknown"
)

// String returns the string representation of the entity type
func (e EntityType) String() string {
	return string(e)
}

// Enum value maps for Entity.
var (
	entityTypeToPb = map[EntityType]pb.Entity{
		RepositoryEntity:       pb.Entity_ENTITY_REPOSITORIES,
		BuildEnvironmentEntity: pb.Entity_ENTITY_BUILD_ENVIRONMENTS,
		ArtifactEntity:         pb.Entity_ENTITY_ARTIFACTS,
		PullRequestEntity:      pb.Entity_ENTITY_PULL_REQUESTS,
		UnknownEntity:          pb.Entity_ENTITY_UNSPECIFIED,
	}
	pbToEntityType = map[pb.Entity]EntityType{
		pb.Entity_ENTITY_REPOSITORIES:       RepositoryEntity,
		pb.Entity_ENTITY_BUILD_ENVIRONMENTS: BuildEnvironmentEntity,
		pb.Entity_ENTITY_ARTIFACTS:          ArtifactEntity,
		pb.Entity_ENTITY_PULL_REQUESTS:      PullRequestEntity,
		pb.Entity_ENTITY_UNSPECIFIED:        UnknownEntity,
	}
)

// IsValidEntity returns true if the entity type is valid
func IsValidEntity(entity pb.Entity) bool {
	switch entity {
	case pb.Entity_ENTITY_REPOSITORIES, pb.Entity_ENTITY_BUILD_ENVIRONMENTS,
		pb.Entity_ENTITY_ARTIFACTS, pb.Entity_ENTITY_PULL_REQUESTS:
		return true
	case pb.Entity_ENTITY_UNSPECIFIED:
		return false
	}
	return false
}

// FromString returns the Entity enum from a string. Typically used in CLI
// when constructing a protobuf message
func FromString(entity string) pb.Entity {
	et := EntityType(strings.ToLower(entity))
	// take advantage of the default value of the map being pb.Entity_ENTITY_UNSPECIFIED
	return entityTypeToPb[et]
}

// KnownTypesCSV returns a comma separated list of known entity types. Useful for UI
func KnownTypesCSV() string {
	var keys []string

	// Iterate through the map and append keys to the slice
	for _, pbval := range pb.Entity_value {
		// PRs are not a first-class object
		if !IsValidEntity(pb.Entity(pbval)) || pb.Entity(pbval) == pb.Entity_ENTITY_PULL_REQUESTS {
			continue
		}
		keys = append(keys, pbToEntityType[pb.Entity(pbval)].String())
	}

	slices.Sort(keys)
	return strings.Join(keys, ",")
}

// EntityTypeFromDB returns the entity type from the database entity
func EntityTypeFromDB(entity db.Entities) pb.Entity {
	switch entity {
	case db.EntitiesRepository:
		return pb.Entity_ENTITY_REPOSITORIES
	case db.EntitiesBuildEnvironment:
		return pb.Entity_ENTITY_BUILD_ENVIRONMENTS
	case db.EntitiesArtifact:
		return pb.Entity_ENTITY_ARTIFACTS
	case db.EntitiesPullRequest:
		return pb.Entity_ENTITY_PULL_REQUESTS
	default:
		return pb.Entity_ENTITY_UNSPECIFIED
	}
}

// EntityTypeToDB returns the database entity from the protobuf entity type
func EntityTypeToDB(entity pb.Entity) db.Entities {
	var dbEnt db.Entities

	switch entity {
	case pb.Entity_ENTITY_REPOSITORIES:
		dbEnt = db.EntitiesRepository
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		dbEnt = db.EntitiesBuildEnvironment
	case pb.Entity_ENTITY_ARTIFACTS:
		dbEnt = db.EntitiesArtifact
	case pb.Entity_ENTITY_PULL_REQUESTS:
		dbEnt = db.EntitiesPullRequest
	case pb.Entity_ENTITY_UNSPECIFIED:
		// This shouldn't happen
	}

	return dbEnt
}
