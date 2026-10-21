// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		wantErr      string
	}{
		{
			name:         "DockerHub tagged image",
			artifactName: "alpine:latest",
			namespace:    "library",
			registry:     "docker.io",
			wantRepo:     "library/alpine",
		},
		{
			name:         "untagged image",
			artifactName: "alpine",
			namespace:    "library",
			registry:     "docker.io",
			wantRepo:     "library/alpine",
		},
		{
			name:         "Quay image",
			artifactName: "envoy:v1.30",
			namespace:    "myorg",
			registry:     "quay.io",
			wantRepo:     "myorg/envoy",
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

			props, err := NewArtifactProperties(tt.artifactName, tt.namespace, tt.registry, "upstream-id")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, props)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.artifactName, props.GetProperty(properties.PropertyName).GetString())
			assert.Equal(t, "upstream-id", props.GetProperty(properties.PropertyUpstreamID).GetString())
			assert.Equal(t, "container", props.GetProperty(properties.ArtifactPropertyType).GetString())
			assert.Equal(t, tt.wantRepo, props.GetProperty(ArtifactPropertyRepository).GetString())
			assert.Equal(t, tt.namespace, props.GetProperty(ArtifactPropertyNamespace).GetString())
			assert.Equal(t, tt.registry, props.GetProperty(ArtifactPropertyRegistry).GetString())
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
		wantErr   string
	}{
		{
			name:      "tagged name drops the tag",
			props:     mustArtifactProperties(t, "myimage:v1.2"),
			namespace: "testns",
			wantName:  "myimage",
		},
		{
			name:      "untagged name is unchanged",
			props:     mustArtifactProperties(t, "myimage"),
			namespace: "testns",
			wantName:  "myimage",
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
			assert.Equal(t, "myimage@sha256:abc", got.GetArtifactPk())
			assert.Equal(t, tt.namespace, got.GetOwner())
			assert.Equal(t, tt.wantName, got.GetName())
			assert.Equal(t, "container", got.GetType())
		})
	}
}

func mustArtifactProperties(t *testing.T, name string) *properties.Properties {
	t.Helper()
	props, err := NewArtifactProperties(name, "testns", "docker.io", "myimage@sha256:abc")
	require.NoError(t, err)
	return props
}
