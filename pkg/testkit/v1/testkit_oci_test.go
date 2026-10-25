// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	artif "github.com/mindersec/minder/internal/providers/artifact"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

const (
	indexManifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[]}`
	// OCI image manifests may omit mediaType.
	imageManifest   = `{"schemaVersion":2,"config":{},"layers":[]}`
	untypedIndex    = `{"schemaVersion":2,"manifests":[]}`
	dockerMediaType = "application/vnd.docker.distribution.manifest.v2+json"
)

func digestOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}

var testArtifact = &minderv1.Artifact{Owner: "stacklok", Name: "minder", Type: "container"}

func TestOCINoFixture(t *testing.T) {
	t.Parallel()

	tk := tkv1.NewTestKit()
	ctx := context.Background()

	_, err := tk.GetArtifactVersions(ctx, testArtifact, nil)
	require.ErrorIs(t, err, tkv1.ErrNoArtifactFixture)

	_, err = tk.GetRawManifest(ctx, "minder", digestOf(indexManifest))
	require.ErrorIs(t, err, tkv1.ErrNoArtifactFixture)

	_, err = tk.GetDigest(ctx, "minder", "latest")
	require.ErrorIs(t, err, tkv1.ErrNoArtifactFixture)

	assert.Empty(t, tk.GetRegistry())
}

func TestGetArtifactVersionsFilter(t *testing.T) {
	t.Parallel()

	old := time.Now().AddDate(-1, 0, 0)
	fixture := tkv1.ArtifactFixture{
		Versions: []tkv1.ArtifactVersionFixture{
			{Tags: []string{"v1.0.0"}, Manifest: []byte(indexManifest)},
			{Tags: []string{"v0.9.0"}, Manifest: []byte(imageManifest)},
			{Tags: []string{"v0.1.0"}, Digest: "sha256:old", CreatedAt: old},
			{Tags: []string{"sha256-abc.sig"}, Digest: "sha256:sig"},
		},
	}

	tests := []struct {
		name     string
		tags     []string
		tagRegex string
		want     []string
	}{
		{
			name: "tag list matches one version",
			tags: []string{"v1.0.0"},
			want: []string{digestOf(indexManifest)},
		},
		{
			name:     "regex matches recent versions only",
			tagRegex: "^v",
			want:     []string{digestOf(indexManifest), digestOf(imageManifest)},
		},
		{
			name: "version older than retention is skipped",
			tags: []string{"v0.1.0"},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter, err := artif.BuildFilter(tt.tags, tt.tagRegex)
			require.NoError(t, err)

			tk := tkv1.NewTestKit(tkv1.WithArtifact(fixture))
			versions, err := tk.GetArtifactVersions(context.Background(), testArtifact, filter)
			require.NoError(t, err)

			got := make([]string, 0, len(versions))
			for _, v := range versions {
				got = append(got, v.GetSha())
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetArtifactVersionsFields(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
		Versions: []tkv1.ArtifactVersionFixture{
			{Tags: []string{"latest", "v1"}, Manifest: []byte(indexManifest), CreatedAt: created},
			{Tags: []string{"v2"}, Digest: "sha256:abcd"},
		},
	}))

	before := time.Now()
	versions, err := tk.GetArtifactVersions(context.Background(), testArtifact, nil)
	require.NoError(t, err)
	require.Len(t, versions, 2)

	assert.Equal(t, []string{"latest", "v1"}, versions[0].GetTags())
	assert.Equal(t, digestOf(indexManifest), versions[0].GetSha())
	assert.True(t, created.Equal(versions[0].GetCreatedAt().AsTime()))

	assert.Equal(t, "sha256:abcd", versions[1].GetSha())
	assert.False(t, versions[1].GetCreatedAt().AsTime().Before(before),
		"created_at should default to the current time")
}

func TestArtifactFixtureErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version tkv1.ArtifactVersionFixture
		errMsg  string
	}{
		{
			name: "digest does not match manifest",
			version: tkv1.ArtifactVersionFixture{
				Tags: []string{"v1"}, Manifest: []byte(indexManifest), Digest: "sha256:wrong",
			},
			errMsg: "does not match manifest content digest",
		},
		{
			name:    "neither manifest nor digest",
			version: tkv1.ArtifactVersionFixture{Tags: []string{"v1"}},
			errMsg:  "either a manifest or a digest is required",
		},
		{
			name:    "manifest is not JSON",
			version: tkv1.ArtifactVersionFixture{Tags: []string{"v1"}, Manifest: []byte("not json")},
			errMsg:  "parsing manifest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
				Versions: []tkv1.ArtifactVersionFixture{tt.version},
			}))
			_, err := tk.GetArtifactVersions(context.Background(), testArtifact, nil)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func TestGetRawManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		version       tkv1.ArtifactVersionFixture
		wantMediaType string
	}{
		{
			name:          "media type from manifest",
			version:       tkv1.ArtifactVersionFixture{Tags: []string{"v1"}, Manifest: []byte(indexManifest)},
			wantMediaType: "application/vnd.oci.image.index.v1+json",
		},
		{
			name:          "untyped index detected by manifests list",
			version:       tkv1.ArtifactVersionFixture{Tags: []string{"v1"}, Manifest: []byte(untypedIndex)},
			wantMediaType: "application/vnd.oci.image.index.v1+json",
		},
		{
			name:          "untyped image manifest",
			version:       tkv1.ArtifactVersionFixture{Tags: []string{"v1"}, Manifest: []byte(imageManifest)},
			wantMediaType: "application/vnd.oci.image.manifest.v1+json",
		},
		{
			name: "media type override",
			version: tkv1.ArtifactVersionFixture{
				Tags: []string{"v1"}, Manifest: []byte(imageManifest), MediaType: dockerMediaType,
			},
			wantMediaType: dockerMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
				Versions: []tkv1.ArtifactVersionFixture{tt.version},
			}))
			content := string(tt.version.Manifest)

			m, err := tk.GetRawManifest(context.Background(), "minder", digestOf(content))
			require.NoError(t, err)
			assert.Equal(t, content, string(m.Content))
			assert.Equal(t, tt.wantMediaType, string(m.MediaType))
			assert.Equal(t, digestOf(content), m.Digest.String())
			assert.Equal(t, int64(len(content)), m.Size)
		})
	}
}

func TestGetRawManifestNotFound(t *testing.T) {
	t.Parallel()

	tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
		Versions: []tkv1.ArtifactVersionFixture{
			{Tags: []string{"v1"}, Manifest: []byte(indexManifest)},
			// A digest-only version has no manifest to return.
			{Tags: []string{"v2"}, Digest: "sha256:nomanifest"},
		},
	}))
	ctx := context.Background()

	_, err := tk.GetRawManifest(ctx, "minder", "sha256:unknown")
	require.ErrorIs(t, err, tkv1.ErrArtifactVersionNotFound)

	_, err = tk.GetRawManifest(ctx, "minder", "sha256:nomanifest")
	require.ErrorIs(t, err, tkv1.ErrArtifactVersionNotFound)
}

func TestGetRawManifestReturnsCopy(t *testing.T) {
	t.Parallel()

	tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
		Versions: []tkv1.ArtifactVersionFixture{{Tags: []string{"v1"}, Manifest: []byte(indexManifest)}},
	}))
	ctx := context.Background()

	m, err := tk.GetRawManifest(ctx, "minder", digestOf(indexManifest))
	require.NoError(t, err)
	m.Content[0] = 'X'

	m, err = tk.GetRawManifest(ctx, "minder", digestOf(indexManifest))
	require.NoError(t, err)
	assert.Equal(t, indexManifest, string(m.Content))
}

func TestGetDigest(t *testing.T) {
	t.Parallel()

	tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{
		Versions: []tkv1.ArtifactVersionFixture{
			{Tags: []string{"latest", "v1"}, Manifest: []byte(indexManifest)},
		},
	}))
	ctx := context.Background()

	digest, err := tk.GetDigest(ctx, "minder", "v1")
	require.NoError(t, err)
	assert.Equal(t, digestOf(indexManifest), digest)

	_, err = tk.GetDigest(ctx, "minder", "missing")
	require.ErrorIs(t, err, tkv1.ErrArtifactVersionNotFound)
}

func TestOCIRegistryAndAuth(t *testing.T) {
	t.Parallel()

	tk := tkv1.NewTestKit(tkv1.WithArtifact(tkv1.ArtifactFixture{Registry: "ghcr.io"}))
	assert.Equal(t, "ghcr.io", tk.GetRegistry())

	auth, err := tk.GetAuthenticator()
	require.NoError(t, err)
	assert.Equal(t, authn.Anonymous, auth)

	_, err = tk.GetManifest(context.Background(), "minder", "v1")
	require.Error(t, err)
	_, err = tk.GetReferrer(context.Background(), "minder", "v1", "")
	require.Error(t, err)
}

func TestArtifactVersionFixtureResolved(t *testing.T) {
	t.Parallel()

	v, err := tkv1.ArtifactVersionFixture{Tags: []string{"v1"}, Manifest: []byte(untypedIndex)}.Resolved()
	require.NoError(t, err)
	assert.Equal(t, digestOf(untypedIndex), v.Digest)
	assert.Equal(t, "application/vnd.oci.image.index.v1+json", v.MediaType)
	assert.True(t, v.CreatedAt.IsZero(), "Resolved leaves created_at to be defaulted at use")

	v, err = tkv1.ArtifactVersionFixture{Tags: []string{"v1"}, Digest: "sha256:abcd"}.Resolved()
	require.NoError(t, err)
	assert.Equal(t, "sha256:abcd", v.Digest)
	assert.Empty(t, v.MediaType)

	_, err = tkv1.ArtifactVersionFixture{Tags: []string{"v1"}}.Resolved()
	require.ErrorContains(t, err, "either a manifest or a digest is required")
}
