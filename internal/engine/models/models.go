// Copyright 2024 Stacklok, Inc.
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

// Package models contains domain models used by the engine
package models

import pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"

// DependencyEcosystem represents an enum of dependency languages
type DependencyEcosystem string

// Enumerated values of DependencyEcosystem
const (
	NPMDependency  DependencyEcosystem = "npm"
	GoDependency   DependencyEcosystem = "go"
	PyPIDependency DependencyEcosystem = "pypi"
)

// Dependency represents a package
type Dependency struct {
	Ecosystem DependencyEcosystem
	Name      string
	Version   string
}

// FilePatch represents the patch which introduced a dependency
type FilePatch struct {
	Name     string
	PatchURL string
}

// ContextualDependency represents a dependency along with where it was imported
type ContextualDependency struct {
	Dep  Dependency
	File FilePatch
}

// PRDependencies represents the dependencies introduced in a PR
type PRDependencies struct {
	PR   *pb.PullRequest
	Deps []ContextualDependency
}

// PRFileLine represents a changed line in a file in a PR
type PRFileLine struct {
	// Deliberately left as an int32: a diff with more than 2^31 lines
	// could lead to various problems while processing.
	LineNumber int32
	Content    string
}

// PRFile represents a file within a PR
type PRFile struct {
	Name         string
	FilePatchURL string
	PatchLines   []PRFileLine
}

// PRContents represents a PR and its changes
type PRContents struct {
	PR    *pb.PullRequest
	Files []PRFile
}
