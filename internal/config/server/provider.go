//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
