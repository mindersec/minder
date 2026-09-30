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

	props := NewArtifactProperties("myimage:v1.2", "myimage@sha256:abc")

	for _, key := range []string{
		properties.PropertyName,
		properties.PropertyUpstreamID,
		properties.ArtifactPropertyType,
	} {
		require.NotNil(t, props.GetProperty(key), "missing property %q", key)
	}

	assert.Equal(t, "myimage:v1.2", props.GetProperty(properties.PropertyName).GetString())
	assert.Equal(t, "myimage@sha256:abc", props.GetProperty(properties.PropertyUpstreamID).GetString())

	typ, err := props.GetProperty(properties.ArtifactPropertyType).AsString()
	require.NoError(t, err)
	assert.Equal(t, "container", typ)
}
