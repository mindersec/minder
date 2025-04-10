// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package properties provides utility functions for fetching and managing properties
package properties

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestNewArtifactFetcher(t *testing.T) {
	t.Parallel()
	fetcher := NewArtifactFetcher()
	assert.NotNil(t, fetcher)
	assert.Len(t, fetcher.propertyOrigins, 1)
	assert.Len(t, fetcher.propertyOrigins[0].keys, 11)
	// all entities should have these properties
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyName)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyUpstreamID)
	// all artifacts should have these properties
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.ArtifactPropertyType)
	// we don't really test for the GH specific properties here
	assert.Empty(t, fetcher.operationalProperties)
}

func TestParseArtifactName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedOwner  string
		expectedName   string
		expectedType   string
		expectedErrMsg string
	}{
		{
			name:          "Valid input with owner",
			input:         "owner/artifact",
			expectedOwner: "owner",
			expectedName:  "artifact",
			expectedType:  "container",
		},
		{
			name:          "Valid input without owner",
			input:         "artifact",
			expectedOwner: "",
			expectedName:  "artifact",
			expectedType:  "container",
		},
		{
			name:           "Invalid input with empty owner",
			input:          "/artifact",
			expectedErrMsg: "invalid name format",
		},
		{
			name:           "Invalid input with empty name",
			input:          "owner/",
			expectedErrMsg: "invalid name format",
		},
		{
			name:          "Invalid input with multiple slashes",
			input:         "owner/artifact/extra",
			expectedOwner: "owner",
			expectedName:  "artifact/extra",
			expectedType:  "container",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner, name, artifactType, err := parseArtifactName(tt.input)

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedType, artifactType)
			}
		})
	}
}

func TestGetName(t *testing.T) {
	t.Parallel()

	fetcher := NewArtifactFetcher()
	tests := []struct {
		name           string
		props          map[string]any
		expected       string
		expectedErrMsg string
	}{
		{
			name: "Valid properties",
			props: map[string]any{
				ArtifactPropertyOwner: "owner",
				ArtifactPropertyName:  "artifact",
			},
			expected: "owner/artifact",
		},
		{
			name: "Missing owner",
			props: map[string]any{
				ArtifactPropertyName: "artifact",
			},
			expected: "artifact",
		},
		{
			name: "Missing name",
			props: map[string]any{
				ArtifactPropertyOwner: "owner",
			},
			expectedErrMsg: "failed to get artifact name",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			props := properties.NewProperties(tt.props)

			result, err := fetcher.GetName(props)
			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetNameFromParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		owner    string
		artifact string
		expected string
	}{
		{
			name:     "With owner",
			owner:    "owner",
			artifact: "artifact",
			expected: "owner/artifact",
		},
		{
			name:     "Without owner",
			owner:    "",
			artifact: "artifact",
			expected: "artifact",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getNameFromParams(tt.owner, tt.artifact)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetArtifactWrapperAttrsFromProps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		props          map[string]any
		expectedOwner  string
		expectedName   string
		expectedType   string
		expectedErrMsg string
	}{
		{
			name: "All properties present",
			props: map[string]any{
				ArtifactPropertyOwner: "owner",
				ArtifactPropertyName:  "artifact",
				ArtifactPropertyType:  "container",
			},
			expectedOwner: "owner",
			expectedName:  "artifact",
			expectedType:  "container",
		},
		{
			name: "Using PropertyName",
			props: map[string]any{
				properties.PropertyName: "owner/artifact",
			},
			expectedOwner: "owner",
			expectedName:  "artifact",
			expectedType:  "container",
		},
		{
			name:           "Missing required properties",
			props:          map[string]any{},
			expectedErrMsg: "missing required properties",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			props := properties.NewProperties(tt.props)

			owner, name, pkgType, err := getArtifactWrapperAttrsFromProps(context.Background(), props)
			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedType, pkgType)
			}
		})
	}
}
