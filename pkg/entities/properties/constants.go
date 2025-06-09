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

// Pull Request property keys
const (
	// PullRequestCommitSHA represents the commit SHA of the pull request
	PullRequestCommitSHA = "commit_sha"
	// PullRequestBaseCloneURL represents the clone URL of the base repository
	PullRequestBaseCloneURL = "base_clone_url"
	// PullRequestBaseBranch represents the target branch of the base repository for pull requests
	PullRequestBaseBranch = "base_branch"
	// PullRequestBaseDefaultBranch represents the default branch of the base repository
	PullRequestBaseDefaultBranch = "base_default_branch"
	// PullRequestTargetCloneURL represents the clone URL of the target repository.
	// Where the pull request comes from.
	PullRequestTargetCloneURL = "target_clone_url"
	// PullRequestTargetBranch represents the default branch of the target repository.
	// Where the pull request comes from.
	PullRequestTargetBranch = "target_branch"
	// PullRequestUpstreamURL represents the URL of the pull request in the provider
	PullRequestUpstreamURL = "upstream_url"
)

// Artifact property keys
const (
	// ArtifactPropertyType represents the type of the artifact (e.g 'container')
	ArtifactPropertyType = "type"
)

// Release property keys
const (
	// ReleasePropertyTag represents the release tag name.
	ReleasePropertyTag = "tag"
	// ReleasePropertyBranch represents the release branch
	ReleasePropertyBranch = "branch"
	// ReleaseCommitSHA represents the commit SHA of the release
	ReleaseCommitSHA = "commit_sha"
)
