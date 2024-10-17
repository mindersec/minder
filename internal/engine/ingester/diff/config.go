// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package diff provides the diff rule data ingest engine
package diff

// DependencyEcosystem is the type of dependency ecosystem
type DependencyEcosystem string

const (
	// DepEcosystemNPM is the npm dependency ecosystem
	DepEcosystemNPM DependencyEcosystem = "npm"
	// DepEcosystemGo is the go dependency ecosystem
	DepEcosystemGo DependencyEcosystem = "go"
	// DepEcosystemPyPI is the python dependency ecosystem
	DepEcosystemPyPI DependencyEcosystem = "pypi"
	// DepEcosystemNone is the fallback value
	DepEcosystemNone DependencyEcosystem = ""
)

// EcosystemMapping is the mapping of a dependency ecosystem to a set of files
type EcosystemMapping struct {
	Ecosystem DependencyEcosystem `json:"ecosystem" yaml:"ecosystem" mapstructure:"ecosystem"`
	Files     []string            `json:"files" yaml:"files" mapstructure:"files"`
}
