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
