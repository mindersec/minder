// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package properties provides utility functions for fetching and managing properties
package properties

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestNewOrganizationFetcher(t *testing.T) {
	t.Parallel()
	fetcher := NewOrganizationFetcher()
	assert.NotNil(t, fetcher)
	assert.Len(t, fetcher.propertyOrigins, 1)
	assert.Len(t, fetcher.propertyOrigins[0].keys, 6)
	// all entities should have these properties
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyName)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyUpstreamID)
	// org-specific properties
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.OrgPropertyIsUser)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.OrgPropertyHasOrganizationProjects)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.OrgPropertyCreatedAt)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.OrgPropertyPlanName)
	assert.Empty(t, fetcher.operationalProperties)
}

func TestOrganizationFetcherGetName(t *testing.T) {
	t.Parallel()

	fetcher := NewOrganizationFetcher()
	tests := []struct {
		name           string
		props          map[string]any
		expected       string
		expectedErrMsg string
	}{
		{
			name: "Valid properties with name",
			props: map[string]any{
				properties.PropertyName: "my-org",
			},
			expected: "my-org",
		},
		{
			name:           "Missing name property",
			props:          map[string]any{},
			expectedErrMsg: "missing property",
		},
		{
			name: "Empty name property",
			props: map[string]any{
				properties.PropertyName: "",
			},
			expectedErrMsg: "missing property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			props := properties.NewProperties(tt.props)

			result, err := fetcher.GetName(props)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
