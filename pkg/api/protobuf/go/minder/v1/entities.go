// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	// ReleaseEntity is an entity abstracting a release
	ReleaseEntity EntityType = "release"
	// PipelineRunEntity is an entity abstracting a pipeline run (eg a workflow)
	PipelineRunEntity EntityType = "pipeline_run"
	// TaskRunEntity is an entity abstracting a task run (eg a step)
	TaskRunEntity EntityType = "task_run"
	// BuildEntity is an entity that represents a software build
	BuildEntity EntityType = "build"
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
		ReleaseEntity:          Entity_ENTITY_RELEASE,
		PipelineRunEntity:      Entity_ENTITY_PIPELINE_RUN,
		TaskRunEntity:          Entity_ENTITY_TASK_RUN,
		BuildEntity:            Entity_ENTITY_BUILD,
		UnknownEntity:          Entity_ENTITY_UNSPECIFIED,
	}
	pbToEntityType = map[Entity]EntityType{
		Entity_ENTITY_REPOSITORIES:       RepositoryEntity,
		Entity_ENTITY_BUILD_ENVIRONMENTS: BuildEnvironmentEntity,
		Entity_ENTITY_ARTIFACTS:          ArtifactEntity,
		Entity_ENTITY_PULL_REQUESTS:      PullRequestEntity,
		Entity_ENTITY_RELEASE:            ReleaseEntity,
		Entity_ENTITY_PIPELINE_RUN:       PipelineRunEntity,
		Entity_ENTITY_TASK_RUN:           TaskRunEntity,
		Entity_ENTITY_BUILD:              BuildEntity,
		Entity_ENTITY_UNSPECIFIED:        UnknownEntity,
	}
)

// IsValid returns true if the entity type is valid
func (entity Entity) IsValid() bool {
	switch entity {
	case Entity_ENTITY_REPOSITORIES, Entity_ENTITY_BUILD_ENVIRONMENTS,
		Entity_ENTITY_ARTIFACTS, Entity_ENTITY_PULL_REQUESTS,
		Entity_ENTITY_RELEASE, Entity_ENTITY_PIPELINE_RUN,
		Entity_ENTITY_TASK_RUN, Entity_ENTITY_BUILD:
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
