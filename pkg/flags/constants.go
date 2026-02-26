// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package flags

const (
	// DockerHubProvider enables the DockerHub provider.
	DockerHubProvider Experiment = "dockerhub_provider"
	// GitLabProvider enables the GitLab provider.
	GitLabProvider Experiment = "gitlab_provider"
	// AlternateMessageDriver enables an an alternate message driver.
	AlternateMessageDriver Experiment = "alternate_message_driver"
	// GitPRDiffs enables the git ingester for pull requests.
	GitPRDiffs Experiment = "git_pr_diffs"
	// DependencyExtract enables functions to perform dependency extraction.
	DependencyExtract Experiment = "dependency_extract"
	// ProjectCreateDelete enables creating top-level projects and deleting them.
	ProjectCreateDelete Experiment = "project_create_delete"
	// AuthenticatedDataSources enables provider authentication for data sources.
	AuthenticatedDataSources Experiment = "authenticated_datasources"
)
