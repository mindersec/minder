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

package v1

import "strings"

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
	entityTypeToPb = map[EntityType]Entity{
		RepositoryEntity:       Entity_ENTITY_REPOSITORIES,
		BuildEnvironmentEntity: Entity_ENTITY_BUILD_ENVIRONMENTS,
		ArtifactEntity:         Entity_ENTITY_ARTIFACTS,
		PullRequestEntity:      Entity_ENTITY_PULL_REQUESTS,
		UnknownEntity:          Entity_ENTITY_UNSPECIFIED,
	}
	pbToEntityType = map[Entity]EntityType{
		Entity_ENTITY_REPOSITORIES:       RepositoryEntity,
		Entity_ENTITY_BUILD_ENVIRONMENTS: BuildEnvironmentEntity,
		Entity_ENTITY_ARTIFACTS:          ArtifactEntity,
		Entity_ENTITY_PULL_REQUESTS:      PullRequestEntity,
		Entity_ENTITY_UNSPECIFIED:        UnknownEntity,
	}
)

// IsValid returns true if the entity type is valid
func (entity Entity) IsValid() bool {
	switch entity {
	case Entity_ENTITY_REPOSITORIES, Entity_ENTITY_BUILD_ENVIRONMENTS,
		Entity_ENTITY_ARTIFACTS, Entity_ENTITY_PULL_REQUESTS:
		return true
	case Entity_ENTITY_UNSPECIFIED:
		return false
	}
	return false
}

// ToString returns the string representation of the entity type
func (entity Entity) ToString() string {
	t, ok := pbToEntityType[entity]
	if !ok {
		return UnknownEntity.String()
	}

	return t.String()
}

// EntityFromString returns the Entity enum from a string. Typically used in CLI
// when constructing a protobuf message
func EntityFromString(entity string) Entity {
	et := EntityType(strings.ToLower(entity))
	// take advantage of the default value of the map being Entity_ENTITY_UNSPECIFIED
	return entityTypeToPb[et]
}

// GetTypeLower returns the type of the artifact entity in lowercase
func (a *Artifact) GetTypeLower() string {
	return strings.ToLower(a.GetType())
}
