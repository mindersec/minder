// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestNewReleaseFetcher(t *testing.T) {
	t.Parallel()

	fetcher := NewReleaseFetcher()
	require.NotNil(t, fetcher)
	assert.Len(t, fetcher.propertyOrigins, 1)
	assert.Len(t, fetcher.propertyOrigins[0].keys, 6)
	assert.Empty(t, fetcher.OperationalProperties())

	// Verify key definitions
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyName)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyUpstreamID)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.ReleasePropertyTag)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.ReleasePropertyBranch)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, ReleasePropertyOwner)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, ReleasePropertyRepo)

	// Verify wrapper lookup
	wrapper := fetcher.WrapperForProperty(ReleasePropertyOwner)
	assert.NotNil(t, wrapper)
	assert.Nil(t, fetcher.WrapperForProperty("non_existent_key"))

	allWrappers := fetcher.AllPropertyWrappers()
	assert.Len(t, allWrappers, 1)
}

func TestReleaseFetcher_GetName(t *testing.T) {
	t.Parallel()

	fetcher := NewReleaseFetcher()

	tests := []struct {
		name         string
		props        map[string]any
		expectedName string
		expectedErr  string
	}{
		{
			name: "valid release with owner, repo, tag",
			props: map[string]any{
				ReleasePropertyOwner:          "mindersec",
				ReleasePropertyRepo:           "minder",
				properties.ReleasePropertyTag: "v1.0.0",
			},
			expectedName: "mindersec/minder/v1.0.0",
		},
		{
			name: "valid release without owner",
			props: map[string]any{
				ReleasePropertyOwner:          "",
				ReleasePropertyRepo:           "minder",
				properties.ReleasePropertyTag: "v1.0.0",
			},
			expectedName: "minder/v1.0.0",
		},
		{
			name: "missing repo property",
			props: map[string]any{
				ReleasePropertyOwner:          "mindersec",
				properties.ReleasePropertyTag: "v1.0.0",
			},
			expectedErr: "failed to get repo name",
		},
		{
			name: "missing tag property",
			props: map[string]any{
				ReleasePropertyOwner: "mindersec",
				ReleasePropertyRepo:  "minder",
			},
			expectedErr: "failed to get tag name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := properties.NewProperties(tt.props)

			name, err := fetcher.GetName(p)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedName, name)
			}
		})
	}
}

func TestGetReleaseNameFromParams(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "mindersec/minder/v1.0.0", getReleaseNameFromParams("mindersec", "minder", "v1.0.0"))
	assert.Equal(t, "minder/v1.0.0", getReleaseNameFromParams("", "minder", "v1.0.0"))
}

func TestNewPropertyFetcherFactory(t *testing.T) {
	t.Parallel()

	factory := NewPropertyFetcherFactory()
	require.NotNil(t, factory)

	prFetcher := factory.EntityPropertyFetcher(minderv1.Entity_ENTITY_PULL_REQUESTS)
	assert.NotNil(t, prFetcher)
	assert.IsType(t, &PullRequestFetcher{}, prFetcher)

	repoFetcher := factory.EntityPropertyFetcher(minderv1.Entity_ENTITY_REPOSITORIES)
	assert.NotNil(t, repoFetcher)
	assert.IsType(t, &RepositoryFetcher{}, repoFetcher)

	artifactFetcher := factory.EntityPropertyFetcher(minderv1.Entity_ENTITY_ARTIFACTS)
	assert.NotNil(t, artifactFetcher)
	assert.IsType(t, &ArtifactFetcher{}, artifactFetcher)

	releaseFetcher := factory.EntityPropertyFetcher(minderv1.Entity_ENTITY_RELEASE)
	assert.NotNil(t, releaseFetcher)
	assert.IsType(t, &ReleaseFetcher{}, releaseFetcher)

	unsupportedFetcher := factory.EntityPropertyFetcher(minderv1.Entity_ENTITY_UNSPECIFIED)
	assert.Nil(t, unsupportedFetcher)
}
