// Copyright 2023 Stacklok, Inc
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

// Package verifyif provides the interface for artifact verifiers, including
// the Result type
package verifyif

import (
	"context"

	"github.com/sigstore/sigstore-go/pkg/verify"
)

// ArtifactRegistry supported artifact registries
type ArtifactRegistry string

const (
	// ArtifactRegistryGHCR is the GitHub Container Registry
	ArtifactRegistryGHCR ArtifactRegistry = "ghcr.io"
)

// ArtifactType represents the type of artifact, i.e., container, npm, etc.
type ArtifactType string

const (
	// ArtifactTypeContainer is a container artifact
	ArtifactTypeContainer ArtifactType = "container"
)

// Result is the result of the verification
type Result struct {
	IsSigned   bool `json:"is_signed"`
	IsVerified bool `json:"is_verified"`
	verify.VerificationResult
}

// ArtifactVerifier is the interface for artifact verifiers
type ArtifactVerifier interface {
	Verify(ctx context.Context, artifactType ArtifactType, registry ArtifactRegistry,
		owner, name, version string) ([]Result, error)
	VerifyContainer(ctx context.Context,
		registry, owner, artifact, version string) ([]Result, error)
	ClearCache()
}
