// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/rs/zerolog"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/verifier/verifyif"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
)

var (
	// ErrNoArtifactFixture is returned by the OCI methods when the TestKit
	// was not configured with WithArtifact.
	ErrNoArtifactFixture = errors.New("no artifact fixture configured in TestKit")

	// ErrArtifactVersionNotFound is returned when no fixture version matches
	// the requested digest or tag.
	ErrArtifactVersionNotFound = errors.New("artifact version not found in fixture")
)

// Ensure that TestKit implements the OCI interface
var _ provv1.OCI = &TestKit{}

// ArtifactFixture describes a single container artifact served by the
// TestKit's OCI methods. The same fixture is returned regardless of the
// artifact name being queried.
type ArtifactFixture struct {
	// Registry is the registry hostname returned by GetRegistry, e.g. "ghcr.io".
	Registry string
	// Versions are the artifact versions available in the registry.
	Versions []ArtifactVersionFixture
}

// ArtifactVersionFixture describes one version of an artifact.
type ArtifactVersionFixture struct {
	// Tags are the tags pointing at this version.
	Tags []string
	// Manifest is the raw manifest or image index, byte for byte as returned
	// by the registry (e.g. saved with curl). If empty, GetRawManifest
	// returns ErrArtifactVersionNotFound for this version.
	Manifest []byte
	// MediaType overrides the media type read from the manifest's
	// mediaType field.
	MediaType string
	// Digest is the version's digest. If Manifest is set, the digest is
	// computed from it, and a non-empty Digest must match. Digest is
	// required when Manifest is empty.
	Digest string
	// CreatedAt is the version's creation time. Defaults to the current time,
	// so fixtures don't fall outside the artifact retention period as they age.
	CreatedAt time.Time
	// Verification holds the sigstore verification results for this
	// version, one per signature or attestation. If empty, the version is
	// treated as unsigned.
	Verification []VerificationFixture
}

// VerificationFixture is a canned sigstore verification result for an
// artifact version.
type VerificationFixture struct {
	// IsSigned is true if a signature or attestation was found.
	IsSigned bool
	// IsVerified is true if the signature or attestation was verified.
	IsVerified bool
	// Result holds the certificate summary and in-toto statement. It may be
	// nil for unsigned or unverified versions.
	Result *verify.VerificationResult
}

// Validate reports whether the fixture is a combination a real verifier
// could return. A verified result must be signed and carry a signing
// certificate, since the verifier only reports one after checking it.
func (f VerificationFixture) Validate() error {
	if !f.IsVerified {
		return nil
	}
	if !f.IsSigned {
		return errors.New("a verified result must also be signed")
	}
	if f.Result == nil || f.Result.Signature == nil || f.Result.Signature.Certificate == nil {
		return errors.New("a verified result must include a signing certificate")
	}
	return nil
}

// Resolved returns a copy of the version with Digest and MediaType filled in
// from Manifest. It returns an error if neither Manifest nor Digest is set,
// if Digest does not match Manifest, or if Manifest is not valid JSON.
func (v ArtifactVersionFixture) Resolved() (ArtifactVersionFixture, error) {
	if len(v.Manifest) == 0 {
		if v.Digest == "" {
			return ArtifactVersionFixture{}, errors.New("either a manifest or a digest is required")
		}
		return v, nil
	}

	hash, _, err := containerv1.SHA256(bytes.NewReader(v.Manifest))
	if err != nil {
		return ArtifactVersionFixture{}, fmt.Errorf("hashing manifest: %w", err)
	}
	if v.Digest != "" && v.Digest != hash.String() {
		return ArtifactVersionFixture{}, fmt.Errorf(
			"digest %s does not match manifest content digest %s", v.Digest, hash)
	}
	v.Digest = hash.String()

	if v.MediaType == "" {
		v.MediaType, err = detectMediaType(v.Manifest)
		if err != nil {
			return ArtifactVersionFixture{}, err
		}
	}
	return v, nil
}

// WithArtifact is a functional option to configure the TestKit to serve
// the given artifact through the OCI provider interface.
func WithArtifact(fixture ArtifactFixture) Option {
	return func(tk *TestKit) {
		tk.artifact = &fixture
	}
}

// resolvedVersion is an ArtifactVersionFixture with its digest and
// descriptor computed.
type resolvedVersion struct {
	tags      []string
	digest    string
	createdAt time.Time
	manifest  *provv1.RawManifest
	// verification is what the TestKit verifier returns for this digest.
	verification []verifyif.Result
}

func (tk *TestKit) resolveArtifactVersions() ([]resolvedVersion, error) {
	if tk.artifact == nil {
		return nil, ErrNoArtifactFixture
	}

	out := make([]resolvedVersion, 0, len(tk.artifact.Versions))
	for i, v := range tk.artifact.Versions {
		rv, err := resolveVersion(v)
		if err != nil {
			return nil, fmt.Errorf("artifact fixture version %d (tags %v): %w", i, v.Tags, err)
		}
		out = append(out, rv)
	}
	return out, nil
}

func resolveVersion(v ArtifactVersionFixture) (resolvedVersion, error) {
	v, err := v.Resolved()
	if err != nil {
		return resolvedVersion{}, err
	}

	verification, err := verificationResults(v.Verification)
	if err != nil {
		return resolvedVersion{}, err
	}

	rv := resolvedVersion{
		tags:         slices.Clone(v.Tags),
		digest:       v.Digest,
		createdAt:    v.CreatedAt,
		verification: verification,
	}
	if rv.createdAt.IsZero() {
		rv.createdAt = time.Now()
	}
	if len(v.Manifest) == 0 {
		return rv, nil
	}

	hash, err := containerv1.NewHash(v.Digest)
	if err != nil {
		return resolvedVersion{}, fmt.Errorf("parsing digest: %w", err)
	}
	rv.manifest = &provv1.RawManifest{
		Descriptor: containerv1.Descriptor{
			MediaType: types.MediaType(v.MediaType),
			Size:      int64(len(v.Manifest)),
			Digest:    hash,
		},
		Content: slices.Clone(v.Manifest),
	}
	return rv, nil
}

// detectMediaType reads the mediaType field of a manifest. OCI manifests may
// omit it, in which case an image index is recognized by its manifests list.
func detectMediaType(manifest []byte) (string, error) {
	var m struct {
		MediaType string            `json:"mediaType"`
		Manifests []json.RawMessage `json:"manifests"`
	}
	if err := json.Unmarshal(manifest, &m); err != nil {
		return "", fmt.Errorf("parsing manifest: %w", err)
	}
	switch {
	case m.MediaType != "":
		return m.MediaType, nil
	case m.Manifests != nil:
		return string(types.OCIImageIndex), nil
	default:
		return string(types.OCIManifestSchema1), nil
	}
}

// GetArtifactVersions implements the ArtifactProvider interface. It applies
// the given filter to each fixture version, as the real providers do.
func (tk *TestKit) GetArtifactVersions(
	ctx context.Context, artifact *minderv1.Artifact, filter provv1.GetArtifactVersionsFilter,
) ([]*minderv1.ArtifactVersion, error) {
	versions, err := tk.resolveArtifactVersions()
	if err != nil {
		return nil, err
	}

	out := make([]*minderv1.ArtifactVersion, 0, len(versions))
	for _, v := range versions {
		if filter != nil {
			if err := filter.IsSkippable(v.createdAt, v.tags); err != nil {
				zerolog.Ctx(ctx).Debug().Str("component", "testkit").
					Str("name", artifact.GetName()).Strs("tags", v.tags).
					Str("reason", err.Error()).Msg("skipping artifact version")
				continue
			}
		}
		out = append(out, &minderv1.ArtifactVersion{
			Tags:      slices.Clone(v.tags),
			Sha:       v.digest,
			CreatedAt: timestamppb.New(v.createdAt),
		})
	}
	return out, nil
}

// GetRawManifest implements the OCI interface by looking up the fixture
// version with the given digest.
func (tk *TestKit) GetRawManifest(_ context.Context, name, digest string) (*provv1.RawManifest, error) {
	versions, err := tk.resolveArtifactVersions()
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.digest == digest && v.manifest != nil {
			m := *v.manifest
			m.Content = slices.Clone(v.manifest.Content)
			return &m, nil
		}
	}
	return nil, fmt.Errorf("%w: no manifest for %s@%s", ErrArtifactVersionNotFound, name, digest)
}

// GetDigest implements the OCI interface by looking up the fixture version
// with the given tag.
func (tk *TestKit) GetDigest(_ context.Context, name, tag string) (string, error) {
	versions, err := tk.resolveArtifactVersions()
	if err != nil {
		return "", err
	}
	for _, v := range versions {
		if slices.Contains(v.tags, tag) {
			return v.digest, nil
		}
	}
	return "", fmt.Errorf("%w: no version tagged %s:%s", ErrArtifactVersionNotFound, name, tag)
}

// GetReferrer implements the OCI interface. It is not supported by TestKit.
func (*TestKit) GetReferrer(_ context.Context, _, _, _ string) (any, error) {
	return nil, errors.New("GetReferrer is not supported in TestKit")
}

// GetManifest implements the OCI interface. It is not supported by TestKit;
// use GetRawManifest instead.
func (*TestKit) GetManifest(_ context.Context, _, _ string) (*containerv1.Manifest, error) {
	return nil, errors.New("GetManifest is not supported in TestKit; use GetRawManifest instead")
}

// GetRegistry implements the OCI interface.
func (tk *TestKit) GetRegistry() string {
	if tk.artifact == nil {
		return ""
	}
	return tk.artifact.Registry
}

// ErrNetworkAccess is returned when something tries to use TestKit
// credentials to reach a real registry.
var ErrNetworkAccess = errors.New("TestKit does not contact real registries")

// GetAuthenticator implements the OCI interface. TestKit never talks to a
// registry, so the returned authenticator fails on use. Anything that
// tries to reach a registry with it gets an error instead of silently
// going out to the network.
func (*TestKit) GetAuthenticator() (authn.Authenticator, error) {
	return refusingAuthenticator{}, nil
}

type refusingAuthenticator struct{}

func (refusingAuthenticator) Authorization() (*authn.AuthConfig, error) {
	return nil, ErrNetworkAccess
}
