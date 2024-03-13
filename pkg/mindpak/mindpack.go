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

// Package mindpak abstracts to bundle profiles and rule types together in
// an artifact that allows for easy distribution and upgrade.
package mindpak

import "time"

// HashAlgorithm is a label that indicates a hashing algorithm
type HashAlgorithm string

const (
	// PathProfiles is the name of the directory holding the profiles of a bundle
	PathProfiles = "profiles"

	// PathRuleTypes is the name of the directory holding the rule types of a bundle
	PathRuleTypes = "rule_types"

	// ManifestFileName is the defaul filename for the manifest
	ManifestFileName = "manifest.json"
)

const (
	// SHA256 is the algorith name constant for the manifest and tests
	SHA256 = HashAlgorithm("sha-256")
)

// Metadata is the data describing the bundle
type Metadata struct {
	Name      string     `json:"name,omitempty"`
	Namespace string     `json:"namespace,omitempty"`
	Version   string     `json:"version,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
}

// File captures the name and hashes of a file included in the bundle
type File struct {
	Name   string                   `json:"name,omitempty"`
	Hashes map[HashAlgorithm]string `json:"hashes,omitempty"`
}

// Files is a collection of the files included in the bundle organized by type
type Files struct {
	Profiles  []*File `json:"profiles,omitempty"`
	RuleTypes []*File `json:"ruleTypes,omitempty"`
}
