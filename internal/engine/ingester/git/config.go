// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package git provides the git rule data ingest engine
package git

// IngesterConfig is the profile-provided configuration for the git ingester
// This allows for users to pass in configuration to the ingester
// in different calls as opposed to having to set it in the rule type.
type IngesterConfig struct {
	Branch   string `json:"branch" yaml:"branch" mapstructure:"branch"`
	CloneURL string `json:"clone_url" yaml:"clone_url" mapstructure:"clone_url"`
}
