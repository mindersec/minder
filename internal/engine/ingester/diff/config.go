// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package diff provides the diff rule data ingest engine
package diff

// DependencyEcosystem is the type of dependency ecosystem
type DependencyEcosystem string

const (
	// DepEcosystemNPM is the npm dependency ecosystem
	DepEcosystemNPM DependencyEcosystem = "npm"
	// DepEcosystemGo is the go dependency ecosystem
	DepEcosystemGo DependencyEcosystem = "Go"
	// DepEcosystemNone is the fallback value
	DepEcosystemNone DependencyEcosystem = ""
)

// EcosystemMapping is the mapping of a dependency ecosystem to a set of files
type EcosystemMapping struct {
	Ecosystem DependencyEcosystem `json:"ecosystem" yaml:"ecosystem" mapstructure:"ecosystem"`
	Files     []string            `json:"files" yaml:"files" mapstructure:"files"`
}
