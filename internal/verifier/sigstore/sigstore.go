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
	"embed"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/stacklok/minder/internal/verifier/sigstore/container"
	"github.com/stacklok/minder/internal/verifier/verifyif"
)

//go:embed tufroots
var embeddedTufRoots embed.FS

const (
	// SigstorePublicTrustedRootRepo is the public trusted root repository for sigstore
	SigstorePublicTrustedRootRepo = "tuf-repo-cdn.sigstore.dev"
	// GitHubSigstoreTrustedRootRepo is the GitHub trusted root repository for sigstore
	GitHubSigstoreTrustedRootRepo = "tuf-repo.github.com"
	// LocalCacheDir is the local cache directory for the verifier
	LocalCacheDir = "/tmp/minder-cache"
	// RootTUFPath is the path to the root.json file inside an embedded TUF repository
	rootTUFPath = "root.json"
)

// Sigstore is the sigstore verifier
type Sigstore struct {
	verifier *verify.SignedEntityVerifier
	authOpts []container.AuthMethod
}

var _ verifyif.ArtifactVerifier = (*Sigstore)(nil)

// New creates a new Sigstore verifier
func New(sigstoreTUFRepoURL string, authOpts ...container.AuthMethod) (*Sigstore, error) {
	// Get the sigstore options for the TUF client and the verifier
	tufOpts, opts, err := getSigstoreOptions(sigstoreTUFRepoURL)
	if err != nil {
		return nil, err
	}

	// Get the trusted material - sigstore's trusted_root.json
	trustedMaterial, err := root.FetchTrustedRootWithOptions(tufOpts)
	if err != nil {
		return nil, err
	}

	sev, err := verify.NewSignedEntityVerifier(trustedMaterial, opts...)
	if err != nil {
		return nil, err
	}

	// return the verifier
	return &Sigstore{
		verifier: sev,
		authOpts: authOpts,
	}, nil
}

func getSigstoreOptions(sigstoreTUFRepoURL string) (*tuf.Options, []verify.VerifierOption, error) {
	// Default the sigstoreTUFRepoURL to the sigstore public trusted root repo if not provided
	if sigstoreTUFRepoURL == "" {
		sigstoreTUFRepoURL = SigstorePublicTrustedRootRepo
	}

	// Get the Sigstore TUF client options
	tufOpts, err := getTUFOptions(sigstoreTUFRepoURL)
	if err != nil {
		return nil, nil, err
	}

	// Get the Sigstore verifier options
	opts, err := verifierOptions(sigstoreTUFRepoURL)
	if err != nil {
		return nil, nil, err
	}

	// All good
	return tufOpts, opts, nil
}

func getTUFOptions(sigstoreTUFRepoURL string) (*tuf.Options, error) {
	// Default the TUF options
	tufOpts := tuf.DefaultOptions()
	tufOpts.DisableLocalCache = true

	// Set the repository base URL, fix the scheme if not provided
	tufURL, err := url.Parse(sigstoreTUFRepoURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing sigstore TUF repo URL: %w", err)
	}
	if tufURL.Scheme == "" {
		tufURL.Scheme = "https"
	}
	tufOpts.RepositoryBaseURL = tufURL.String()

	// sigstore-go has a copy of the root.json for the public sigstore instance embedded. Nothing to do.
	if sigstoreTUFRepoURL != SigstorePublicTrustedRootRepo {
		// Look up and set the embedded root.json for the given TUF repository
		rootJson, err := embeddedRootJson(sigstoreTUFRepoURL)
		if err != nil {
			return nil, fmt.Errorf("error getting embedded root.json for %s: %w", sigstoreTUFRepoURL, err)
		}
		tufOpts.Root = rootJson
	}

	// All good
	return tufOpts, nil
}

func verifierOptions(trustedRoot string) ([]verify.VerifierOption, error) {
	switch trustedRoot {
	case SigstorePublicTrustedRootRepo:
		return []verify.VerifierOption{
			verify.WithSignedCertificateTimestamps(1),
			verify.WithTransparencyLog(1),
			verify.WithObserverTimestamps(1),
		}, nil
	case GitHubSigstoreTrustedRootRepo:
		return []verify.VerifierOption{
			verify.WithObserverTimestamps(1),
		}, nil
	}
	return nil, fmt.Errorf("unknown trusted root: %s", trustedRoot)
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

// sanitizeInput sanitizes the input parameters
func sanitizeInput(registry *verifyif.ArtifactRegistry, owner *string) {
	// Default the registry to GHCR for the time being
	if *registry == "" {
		*registry = verifyif.ArtifactRegistryGHCR
	}
	// (jaosorior): The owner can't be upper-cased, normalize the owner.
	*owner = strings.ToLower(*owner)
}

func embeddedRootJson(tufRootURL string) ([]byte, error) {
	embeddedRootPath := path.Join("tufroots", tufRootURL, rootTUFPath)

	return embeddedTufRoots.ReadFile(embeddedRootPath)
}

// ClearCache clears the sigstore cache
func (_ *Sigstore) ClearCache() {
}
