// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// ProviderConfig is the configuration for the providers
type ProviderConfig struct {
	GitHubApp *GitHubAppConfig `mapstructure:"github-app"`
	GitHub    *GitHubConfig    `mapstructure:"github"`
	Git       GitConfig        `mapstructure:"git"`
	GitLab    *GitLabConfig    `mapstructure:"gitlab"`
}

// GitConfig provides server-side configuration for Git operations like "clone"
type GitConfig struct {
	MaxFiles int64 `mapstructure:"max_files" default:"10000"`
	MaxBytes int64 `mapstructure:"max_bytes" default:"100_000_000"`
}
