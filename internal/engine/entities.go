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

package engine

import "github.com/stacklok/mediator/pkg/db"

// EntityType is the type of entity
type EntityType string

// Entity types
const (
	// RepositoryEntity is a repository entity
	RepositoryEntity EntityType = "repository"
	// BuildEnvironmentEntity is a build environment entity
	BuildEnvironmentEntity EntityType = "build_environment"
	// ArtifactEntity is an artifact entity
	ArtifactEntity EntityType = "artifact"
	// UnknownEntity is an explicitly unknown entity
	UnknownEntity EntityType = "unknown"
)

// String returns the string representation of the entity type
func (e EntityType) String() string {
	return string(e)
}

// IsValidEntity returns true if the entity type is valid
func IsValidEntity(entity EntityType) bool {
	switch entity {
	case RepositoryEntity, BuildEnvironmentEntity, ArtifactEntity:
		return true
	}
	return false
}

// EntityTypeFromDB returns the entity type from the database entity
func EntityTypeFromDB(entity db.Entities) EntityType {
	switch entity {
	case db.EntitiesRepository:
		return RepositoryEntity
	case db.EntitiesBuildEnvironment:
		return BuildEnvironmentEntity
	case db.EntitiesArtifact:
		return ArtifactEntity
	default:
		return UnknownEntity
	}
}
