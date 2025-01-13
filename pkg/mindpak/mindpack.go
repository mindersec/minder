// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

	// PathDataSources is the name of the directory holding the data sources of a bundle
	PathDataSources = "data_sources"

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
	Profiles    []*File `json:"profiles,omitempty"`
	RuleTypes   []*File `json:"ruleTypes,omitempty"`
	DataSources []*File `json:"dataSources,omitempty"`
}
