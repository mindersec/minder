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

// Package verifier provides a client for verifying various types of artifacts against various provenance mechanisms
package verifier

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/stacklok/minder/internal/verifier/sigstore"
	"github.com/stacklok/minder/internal/verifier/sigstore/container"
	"github.com/stacklok/minder/internal/verifier/verifyif"
)

const (
	// ArtifactSignatureSuffix is the suffix for the signature tag
	ArtifactSignatureSuffix = ".sig"
	// LocalCacheDir is the local cache directory for the verifier
	LocalCacheDir = "/tmp/minder-cache"
)

// Type represents the type of verifier, i.e., sigstore, slsa, etc.
type Type string

const (
	// VerifierSigstore is the sigstore verifier
	VerifierSigstore Type = "sigstore"
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

// Verifier is the object that verifies artifacts
type Verifier struct {
	verifier verifyif.ArtifactVerifier
	cacheDir string
}

// NewVerifier creates a new Verifier object
func NewVerifier(verifier Type, verifierURL string, containerAuth ...container.AuthMethod) (*Verifier, error) {
	var err error
	var v verifyif.ArtifactVerifier

	// create a temporary directory for storing the sigstore cache
	tmpDir, err := createTmpDir(LocalCacheDir, "sigstore")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary sigstore cache directory: %w", err)
	}
	if verifierURL == "" {
		verifierURL = sigstore.SigstorePublicTrustedRootRepo
	}
	// create the verifier
	switch verifier {
	case VerifierSigstore:
		// Default the verifier URL to the sigstore public trusted root repo
		if verifierURL == "" {
			verifierURL = sigstore.SigstorePublicTrustedRootRepo
		}
		v, err = sigstore.New(verifierURL, tmpDir, containerAuth...)
		if err != nil {
			return nil, fmt.Errorf("error creating sigstore verifier: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown verifier type: %s", verifier)
	}

	// return the verifier
	return &Verifier{
		verifier: v,
		cacheDir: tmpDir,
	}, nil
}

// Verify verifies an artifact
func (v *Verifier) Verify(ctx context.Context, artifactType ArtifactType, registry ArtifactRegistry,
	owner, artifact, version string) (*verifyif.Result, error) {
	var err error
	var res *verifyif.Result
	// Sanitize the input
	sanitizeInput(&registry, &owner)

	// Process verification based on the artifact type
	switch artifactType {
	case ArtifactTypeContainer:
		res, err = v.verifier.VerifyContainer(ctx, string(registry), owner, artifact, version)
	default:
		err = fmt.Errorf("unknown artifact type: %s", artifactType)
	}

	return res, err
}

// ClearCache cleans up the verifier cache directory and all its contents
// This is temporary until sigstore-go supports in-memory verification
func (v *Verifier) ClearCache() {
	if err := os.RemoveAll(v.cacheDir); err != nil {
		log.Err(err).Msg("error deleting temporary sigstore cache directory")
	}
}

// GetSignatureTag returns the signature tag for a given image, if exists, otherwise empty string
func GetSignatureTag(tags []string) string {
	// if the artifact has a .sig tag it's a signature, skip it
	for _, tag := range tags {
		if strings.HasSuffix(tag, ArtifactSignatureSuffix) {
			return tag
		}
	}
	return ""
}

// sanitizeInput sanitizes the input parameters
func sanitizeInput(registry *ArtifactRegistry, owner *string) {
	// Default the registry to GHCR for the time being
	if *registry == "" {
		*registry = ArtifactRegistryGHCR
	}
	// (jaosorior): The owner can't be upper-cased, normalize the owner.
	*owner = strings.ToLower(*owner)
}

func createTmpDir(path, prefix string) (string, error) {
	// ensure the path exists
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to ensure path for temporary sigstore cache directory: %w", err)
	}
	// create the temporary directory
	tmpDir, err := os.MkdirTemp(path, prefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary sigstore cache directory: %w", err)
	}
	return tmpDir, nil
}
