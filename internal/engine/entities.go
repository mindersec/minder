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

// Entity types
const (
	// RepositoryEntity is a repository entity
	RepositoryEntity = "repository"
	// BuildEnvironmentEntity is a build environment entity
	BuildEnvironmentEntity = "build_environment"
	// ArtifactEntity is an artifact entity
	ArtifactEntity = "artifact"
)

// IsValidEntity returns true if the entity type is valid
func IsValidEntity(entity string) bool {
	switch entity {
	case RepositoryEntity, BuildEnvironmentEntity, ArtifactEntity:
		return true
	}
	return false
}
