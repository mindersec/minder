// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestParseImageRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ref      string
		wantRepo string
		wantTag  string
		wantErr  string
	}{
		{
			name:     "repo only defaults tag",
			ref:      "myimage",
			wantRepo: "myimage",
			wantTag:  DefaultTag,
		},
		{
			name:     "repo and tag",
			ref:      "myimage:v1.2",
			wantRepo: "myimage",
			wantTag:  "v1.2",
		},
		{
			name:    "empty repo",
			ref:     ":v1.2",
			wantErr: `invalid image reference ":v1.2": missing repository`,
		},
		{
			name:    "empty ref",
			ref:     "",
			wantErr: `invalid image reference "": missing repository`,
		},
		{
			name:    "empty tag",
			ref:     "myimage:",
			wantErr: `invalid image reference "myimage:": empty tag`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, tag, err := ParseImageRef(tt.ref)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Empty(t, repo)
				assert.Empty(t, tag)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRepo, repo)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

func TestHasExplicitTag(t *testing.T) {
	t.Parallel()

	assert.True(t, HasExplicitTag("alpine:latest"))
	assert.True(t, HasExplicitTag("myorg/alpine:v1.2"))
	assert.False(t, HasExplicitTag("alpine"))
	assert.False(t, HasExplicitTag("myorg/alpine"))
}

func TestArtifactNameFromProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		props   *properties.Properties
		want    string
		wantErr string
	}{
		{
			name:  "name is returned unchanged",
			props: properties.NewProperties(map[string]any{properties.PropertyName: "myimage:v1.2"}),
			want:  "myimage:v1.2",
		},
		{
			name:    "missing name property",
			props:   properties.NewProperties(map[string]any{}),
			wantErr: "failed to get artifact name",
		},
		{
			name:    "empty name property",
			props:   properties.NewProperties(map[string]any{properties.PropertyName: ""}),
			wantErr: "artifact name is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ArtifactNameFromProperties(tt.props)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewArtifactProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		artifactName string
		namespace    string
		registry     string
		wantRepo     string
		wantUpstream string
		wantErr      string
	}{
		{
			name:         "DockerHub tagged image",
			artifactName: "alpine:latest",
			namespace:    "library",
			registry:     "docker.io",
			wantRepo:     "library/alpine",
			wantUpstream: "alpine:latest",
		},
		{
			name:         "untagged image gets the default tag",
			artifactName: "alpine",
			namespace:    "library",
			registry:     "docker.io",
			wantRepo:     "library/alpine",
			wantUpstream: "alpine:latest",
		},
		{
			name:         "Quay image",
			artifactName: "envoy:v1.30",
			namespace:    "myorg",
			registry:     "quay.io",
			wantRepo:     "myorg/envoy",
			wantUpstream: "envoy:v1.30",
		},
		{
			name:         "missing repository",
			artifactName: ":v1.2",
			namespace:    "library",
			registry:     "docker.io",
			wantErr:      "missing repository",
		},
		{
			name:         "empty tag",
			artifactName: "alpine:",
			namespace:    "library",
			registry:     "docker.io",
			wantErr:      "empty tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			props, err := NewArtifactProperties(tt.artifactName, tt.namespace, tt.registry, "sha256:abc")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, props)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.artifactName, props.GetProperty(properties.PropertyName).GetString())
			assert.Equal(t, tt.wantUpstream, props.GetProperty(properties.PropertyUpstreamID).GetString())
			assert.Equal(t, "container", props.GetProperty(properties.ArtifactPropertyType).GetString())
			assert.Equal(t, tt.wantRepo, props.GetProperty(ArtifactPropertyRepository).GetString())
			assert.Equal(t, tt.namespace, props.GetProperty(ArtifactPropertyNamespace).GetString())
			assert.Equal(t, tt.registry, props.GetProperty(ArtifactPropertyRegistry).GetString())
			assert.Equal(t, "sha256:abc", props.GetProperty(ArtifactPropertyDigest).GetString())
		})
	}
}

func TestArtifactV1FromProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		props     *properties.Properties
		namespace string
		wantName  string
		wantPk    string
		wantErr   string
	}{
		{
			name:      "tagged name drops the tag",
			props:     mustArtifactProperties(t, "myimage:v1.2"),
			namespace: "testns",
			wantName:  "myimage",
			wantPk:    "myimage:v1.2",
		},
		{
			name:      "untagged name is unchanged",
			props:     mustArtifactProperties(t, "myimage"),
			namespace: "testns",
			wantName:  "myimage",
			wantPk:    "myimage:latest",
		},
		{
			name:      "missing name",
			props:     properties.NewProperties(map[string]any{}),
			namespace: "testns",
			wantErr:   "failed to get artifact name",
		},
		{
			name: "missing upstream ID",
			props: properties.NewProperties(map[string]any{
				properties.PropertyName: "myimage",
			}),
			namespace: "testns",
			wantErr:   "failed to get artifact upstream ID",
		},
		{
			name:    "empty namespace",
			props:   mustArtifactProperties(t, "myimage"),
			wantErr: "artifact namespace is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ArtifactV1FromProperties(tt.props, tt.namespace)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPk, got.GetArtifactPk())
			assert.Equal(t, tt.namespace, got.GetOwner())
			assert.Equal(t, tt.wantName, got.GetName())
			assert.Equal(t, "container", got.GetType())
		})
	}
}

func mustArtifactProperties(t *testing.T, name string) *properties.Properties {
	t.Helper()
	props, err := NewArtifactProperties(name, "testns", "docker.io", "sha256:abc")
	require.NoError(t, err)
	return props
}

func TestArtifactTag(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		art     *minderv1.Artifact
		wantTag string
		wantOK  bool
	}{
		{name: "repo:tag", art: &minderv1.Artifact{Name: "myimage", ArtifactPk: "myimage:v1.2"},
			wantTag: "v1.2", wantOK: true},
		{name: "nested repo", art: &minderv1.Artifact{Name: "team/app", ArtifactPk: "team/app:latest"},
			wantTag: "latest", wantOK: true},
		{name: "legacy repo@digest", art: &minderv1.Artifact{Name: "myimage", ArtifactPk: "myimage@sha256:abc"}},
		{name: "pk for another repo", art: &minderv1.Artifact{Name: "myimage", ArtifactPk: "other:v1"}},
		{name: "empty tag", art: &minderv1.Artifact{Name: "myimage", ArtifactPk: "myimage:"}},
		{name: "GHCR numeric pk", art: &minderv1.Artifact{Name: "myimage", ArtifactPk: "12345"}},
		{name: "empty name", art: &minderv1.Artifact{ArtifactPk: ":v1"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tag, ok := (&OCI{}).ArtifactTag(tt.art)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}

// TestArtifactTagRoundTrip builds an artifact the way the DockerHub and Quay
// providers do and checks ArtifactTag recovers the registered tag. If the
// upstream ID format changes without ArtifactTag, this fails instead of
// ingestion silently falling back to the repository-wide path.
func TestArtifactTagRoundTrip(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		ref     string
		wantTag string
	}{
		{name: "explicit tag", ref: "myimage:v1.2", wantTag: "v1.2"},
		{name: "default tag", ref: "myimage", wantTag: DefaultTag},
		{name: "nested repo", ref: "team/app:stable", wantTag: "stable"},
		{name: "tag with dots and dashes", ref: "myimage:1.2.3-rc.1", wantTag: "1.2.3-rc.1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			props, err := NewArtifactProperties(tt.ref, "testns", "docker.io", "sha256:abc")
			require.NoError(t, err)
			art, err := ArtifactV1FromProperties(props, "testns")
			require.NoError(t, err)

			tag, ok := (&OCI{}).ArtifactTag(art)
			require.True(t, ok, "ArtifactTag rejected %q (ArtifactPk %q)", tt.ref, art.GetArtifactPk())
			assert.Equal(t, tt.wantTag, tag)
		})
	}
}
