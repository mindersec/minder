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
	// VulnCheckErrorTemplate enables improved evaluation details
	// messages in the vulncheck rule.
	VulnCheckErrorTemplate Experiment = "vulncheck_error_template"
	// AlternateMessageDriver enables an an alternate message driver.
	AlternateMessageDriver Experiment = "alternate_message_driver"
)
