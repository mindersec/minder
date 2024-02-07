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

// Package sigstore provides a client for verifying artifacts using sigstore
package sigstore

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/stacklok/minder/internal/verifier/sigstore/container"
	"github.com/stacklok/minder/internal/verifier/verifyif"
)

const (
	// SigstorePublicTrustedRootRepo is the public trusted root repository for sigstore
	SigstorePublicTrustedRootRepo = "tuf-repo-cdn.sigstore.dev"
	// LocalCacheDir is the local cache directory for the verifier
	LocalCacheDir = "/tmp/minder-cache"
)

// Sigstore is the sigstore verifier
type Sigstore struct {
	verifier *verify.SignedEntityVerifier
	authOpts []container.AuthMethod
	cacheDir string
}

var _ verifyif.ArtifactVerifier = (*Sigstore)(nil)

// New creates a new Sigstore verifier
func New(trustedRoot string, authOpts ...container.AuthMethod) (*Sigstore, error) {
	cacheDir, err := createTmpDir(LocalCacheDir, "sigstore")
	if err != nil {
		return nil, err
	}

	// init sigstore's verifier
	trustedrootJSON, err := tuf.GetTrustedrootJSON(trustedRoot, cacheDir)
	if err != nil {
		return nil, err
	}
	trustedMaterial, err := root.NewTrustedRootFromJSON(trustedrootJSON)
	if err != nil {
		return nil, err
	}
	sev, err := verify.NewSignedEntityVerifier(trustedMaterial, verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1))
	if err != nil {
		return nil, err
	}

	// return the verifier
	return &Sigstore{
		verifier: sev,
		authOpts: authOpts,
		cacheDir: cacheDir,
	}, nil
}

// Verify verifies an artifact
func (s *Sigstore) Verify(ctx context.Context, artifactType verifyif.ArtifactType, registry verifyif.ArtifactRegistry,
	owner, artifact, version string) ([]verifyif.Result, error) {
	var err error
	var res []verifyif.Result
	// Sanitize the input
	sanitizeInput(&registry, &owner)

	// Process verification based on the artifact type
	switch artifactType {
	case verifyif.ArtifactTypeContainer:
		res, err = s.VerifyContainer(ctx, string(registry), owner, artifact, version)
	default:
		err = fmt.Errorf("unknown artifact type: %s", artifactType)
	}

	return res, err
}

// VerifyContainer verifies a container artifact using sigstore
func (s *Sigstore) VerifyContainer(ctx context.Context, registry, owner, artifact, version string) (
	[]verifyif.Result, error) {
	return container.Verify(ctx, s.verifier, registry, owner, artifact, version, s.authOpts...)
}

// ClearCache clears the sigstore cache
func (s *Sigstore) ClearCache() {
	if err := os.RemoveAll(s.cacheDir); err != nil {
		log.Err(err).Msg("error deleting temporary sigstore cache directory")
	}
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

// sanitizeInput sanitizes the input parameters
func sanitizeInput(registry *verifyif.ArtifactRegistry, owner *string) {
	// Default the registry to GHCR for the time being
	if *registry == "" {
		*registry = verifyif.ArtifactRegistryGHCR
	}
	// (jaosorior): The owner can't be upper-cased, normalize the owner.
	*owner = strings.ToLower(*owner)
}
