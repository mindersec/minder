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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
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
	cacheDir string
}

var _ verifyif.ArtifactVerifier = (*Sigstore)(nil)

// New creates a new Sigstore verifier
func New(trustedRoot string, authOpts ...container.AuthMethod) (*Sigstore, error) {
	cacheDir, err := createTmpDir(LocalCacheDir, "sigstore")
	if err != nil {
		return nil, err
	}

	if err := seedRootJson(trustedRoot, cacheDir); err != nil {
		return nil, fmt.Errorf("seeding root: %w", err)
	}

	trustedMaterial, err := readTrustedRoot(trustedRoot, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("reading root: %w", err)
	}

	opts, err := verifierOptions(trustedRoot)
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
		cacheDir: cacheDir,
	}, nil
}

// readTrustedRoot reads a trusted tuf root stored in rootSource. If a cache
// directory is specified, the function will check the directory for a precached
// copy.
func readTrustedRoot(rootSource, cacheDir string) (*root.TrustedRoot, error) {
	var cached []byte
	var err error

	if cacheDir != "" {
		cached, err = readRootJson(rootSource, cacheDir)
		if err != nil {
			return nil, fmt.Errorf("checking cache: %s", err)
		}
	}

	if cached != nil {
		rt, err := root.NewTrustedRootFromJSON(cached)
		if err != nil {
			return nil, fmt.Errorf("creating new root from cached json: %w", err)
		}
		return rt, nil
	}

	tufOpts := tuf.DefaultOptions()

	// Our module keeps its own cache, so we disable
	// the sigstore built in
	tufOpts.DisableLocalCache = true

	if rootSource != "" {
		tufOpts.RepositoryBaseURL = "https://" + rootSource
	}

	trustedMaterial, err := root.FetchTrustedRootWithOptions(tufOpts)
	if err != nil {
		return nil, fmt.Errorf("fetching root: %w", err)
	}

	// (TODO) write the new material to cache

	return trustedMaterial, nil
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

// ClearCache clears the sigstore cache
func (s *Sigstore) ClearCache() {
	if err := os.RemoveAll(s.cacheDir); err != nil {
		log.Err(err).Msg("error deleting temporary sigstore cache directory")
	}
}

func createTmpDir(basePath, prefix string) (string, error) {
	// ensure the path exists
	err := os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to ensure path for temporary sigstore cache directory: %w", err)
	}
	// create the temporary directory
	tmpDir, err := os.MkdirTemp(basePath, prefix)
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

func seedRootJson(tufRepo, cacheDir string) error {
	// sigstore-go has a copy of the root.json for the public sigstore
	// instance embedded. Nothing to do.
	if tufRepo == SigstorePublicTrustedRootRepo {
		return nil
	}

	// check if the repo is one of the well-known embedded TUF repositories
	rootJson, err := embeddedRootJson(tufRepo)
	if err != nil {
		return fmt.Errorf("error getting embedded root.json for %s: %w", tufRepo, err)
	}
	return writeRootJson(tufRepo, cacheDir, rootJson)
}

func embeddedRootJson(tufRootURL string) ([]byte, error) {
	embeddedRootPath := path.Join("tufroots", tufRootURL, rootTUFPath)

	return embeddedTufRoots.ReadFile(embeddedRootPath)
}

// readRootJson reads a cached root from the cache. returns nil
// if there is no match.
func readRootJson(tufRepo, cacheDir string) ([]byte, error) {
	// Don't cache the empty string, this delegates the default
	// to the sigstore-go client
	if tufRepo == "" {
		return nil, nil
	}

	cachedPath := filepath.Join(cacheDir, tufRepo, rootTUFPath)
	cachedPath = filepath.Clean(cachedPath)
	if !strings.HasPrefix(cachedPath, cacheDir) {
		return nil, fmt.Errorf("unsafe cache path when reading")
	}

	_, err := os.Stat(cachedPath)

	// (TODO) Handle cache invalidation here
	if err == nil {
		data, err := os.ReadFile(cachedPath)
		if err != nil {
			return nil, fmt.Errorf("reading cached file: %w", err)
		}
		return data, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("checking local root cache for %q: %s", tufRepo, err)
	}

	return nil, nil
}

func writeRootJson(tufRepo, cacheDir string, rootJson []byte) error {
	const (
		newFilePerms = os.FileMode(0600)
		newDirPerms  = os.FileMode(0750)
	)

	tufPath := path.Join(cacheDir, tufRepo)
	fi, err := os.Stat(tufPath)
	if errors.Is(err, fs.ErrNotExist) {
		if err = os.MkdirAll(tufPath, newDirPerms); err != nil {
			return fmt.Errorf("error creating directory for metadata cache: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error getting FileInfo for %s: %w", tufPath, err)
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("can not open %s, not a directory", tufPath)
		}
		// Verify file mode is not too permissive.
		if err = ensureMaxPermissions(fi, newDirPerms); err != nil {
			return err
		}
	}

	rootPath := path.Join(tufPath, rootTUFPath)
	return os.WriteFile(rootPath, rootJson, newFilePerms)
}

// taken from go-tuf/internal/fsutil/perm.go
//
// EnsureMaxPermissions tests the provided file info, returning an error if the
// file's permission bits contain excess permissions not set in maxPerms.
//
// For example, a file with permissions -rw------- will successfully validate
// with maxPerms -rw-r--r-- or -rw-rw-r--, but will not validate with maxPerms
// -r-------- (due to excess --w------- permission) or --w------- (due to
// excess -r-------- permission).
//
// Only permission bits of the file modes are considered.
func ensureMaxPermissions(fi os.FileInfo, maxPerms os.FileMode) error {
	gotPerm := fi.Mode().Perm()
	forbiddenPerms := (^maxPerms).Perm()
	excessPerms := gotPerm & forbiddenPerms

	if excessPerms != 0 {
		return fmt.Errorf(
			"permission bits for file %v failed validation: want at most %v, got %v with excess perms %v",
			fi.Name(), maxPerms.Perm(), gotPerm, excessPerms)
	}

	return nil
}
