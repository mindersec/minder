// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

// General entity keys
const (
	// PropertyName represents the name of the entity. The name is formatted by the provider
	PropertyName = "name"
	// PropertyUpstreamID represents the ID of the entity in the provider
	PropertyUpstreamID = "upstream_id"
)

// Repository property keys
const (
	// RepoPropertyIsPrivate represents whether the repository is private
	RepoPropertyIsPrivate = "is_private"
	// RepoPropertyIsArchived represents whether the repository is archived
	RepoPropertyIsArchived = "is_archived"
	// RepoPropertyIsFork represents whether the repository is a fork
	RepoPropertyIsFork = "is_fork"
)

// Artifact property keys
const (
	// ArtifactPropertyType represents the type of the artifact (e.g 'container')
	ArtifactPropertyType = "type"
)
