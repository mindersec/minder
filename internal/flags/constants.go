// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package flags

const (
	// UserManagement enables user management, i.e. invitations, role assignments, etc.
	UserManagement Experiment = "user_management"
	// DockerHubProvider enables the DockerHub provider.
	DockerHubProvider Experiment = "dockerhub_provider"
	// GitLabProvider enables the GitLab provider.
	GitLabProvider Experiment = "gitlab_provider"
	// MachineAccounts enables machine accounts (in particular, GitHub Actions) for authorization
	MachineAccounts Experiment = "machine_accounts"
	// VulnCheckErrorTemplate enables improved evaluation details
	// messages in the vulncheck rule.
	VulnCheckErrorTemplate Experiment = "vulncheck_error_template"
	// AlternateMessageDriver enables an an alternate message driver.
	AlternateMessageDriver Experiment = "alternate_message_driver"
	// DataSources enables data sources management.
	DataSources Experiment = "data_sources"
	// PRCommentAlert enables the pull request comment alert engine.
	PRCommentAlert Experiment = "pr_comment_alert"
	// GitPRDiffs enables the git ingester for pull requests.
	GitPRDiffs Experiment = "git_pr_diffs"
	// TarGzFunctions enables functions to produce tar.gz data in the rego
	// evaluation environment.
	TarGzFunctions Experiment = "tar_gz_functions"
	// DependencyExtract enables functions to perform dependency extraction.
	DependencyExtract Experiment = "dependency_extract"
)
