// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// DefaultProfilesConfig holds the profiles installed by default during project
// creation. If omitted - this will default to disabled.
type DefaultProfilesConfig struct {
	Enabled bool `mapstructure:"enabled" default:"false"`
	// List of profile names to install
	Profiles []string `mapstructure:"profiles"`
	// The bundle to subscribe to
	Bundle IncludedBundleConfig `mapstructure:"bundle"`
}

// GetProfiles is a null-safe getter for Profiles
func (d *DefaultProfilesConfig) GetProfiles() []string {
	if d == nil || d.Profiles == nil {
		return []string{}
	}
	return d.Profiles
}

// IncludedBundleConfig holds details about the bundle included with Minder
type IncludedBundleConfig struct {
	Namespace string `mapstructure:"namespace"`
	Name      string `mapstructure:"name"`
}
