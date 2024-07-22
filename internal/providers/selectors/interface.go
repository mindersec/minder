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

// Package selectors provides utilities to convert entities to selector entities.
package selectors

import (
	"context"

	internalpb "github.com/stacklok/minder/internal/proto"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// RepoSelectorConverter is an interface for converting a repository to a repository selector
type RepoSelectorConverter interface {
	provifv1.Provider

	// RepoToSelectorEntity converts the given repository to a repository selector
	RepoToSelectorEntity(ctx context.Context, repo *minderv1.Repository) *internalpb.SelectorEntity
}

// ArtifactSelectorConverter is an interface for converting an artifact to a artifact selector
type ArtifactSelectorConverter interface {
	provifv1.Provider

	// ArtifactToSelectorEntity converts the given artifact to a artifact selector
	ArtifactToSelectorEntity(ctx context.Context, artifact *minderv1.Artifact) *internalpb.SelectorEntity
}
