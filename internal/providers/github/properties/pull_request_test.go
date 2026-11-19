// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestNewPullRequestFetcher(t *testing.T) {
	t.Parallel()

	fetcher := NewPullRequestFetcher()
	require.NotNil(t, fetcher)
	assert.Len(t, fetcher.propertyOrigins, 1)
	assert.Len(t, fetcher.propertyOrigins[0].keys, 20)
	assert.Equal(t, prOperationalProperties, fetcher.OperationalProperties())

	// Verify key definitions
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyName)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyUpstreamID)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, PullPropertyNumber)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, PullPropertyRepoOwner)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, PullPropertyRepoName)

	// Verify wrapper lookup
	wrapper := fetcher.WrapperForProperty(PullPropertyNumber)
	assert.NotNil(t, wrapper)
	assert.Nil(t, fetcher.WrapperForProperty("non_existent_key"))

	allWrappers := fetcher.AllPropertyWrappers()
	assert.Len(t, allWrappers, 1)
}

func TestPullRequestFetcher_GetName(t *testing.T) {
	t.Parallel()

	fetcher := NewPullRequestFetcher()

	tests := []struct {
		name         string
		props        map[string]any
		expectedName string
		expectedErr  string
	}{
		{
			name: "valid pr properties",
			props: map[string]any{
				PullPropertyRepoOwner: "mindersec",
				PullPropertyRepoName:  "minder",
				PullPropertyNumber:    int64(7036),
			},
			expectedName: "mindersec/minder/7036",
		},
		{
			name: "missing owner property",
			props: map[string]any{
				PullPropertyRepoName: "minder",
				PullPropertyNumber:   int64(7036),
			},
			expectedErr: "error fetching pr owner property",
		},
		{
			name: "missing name property",
			props: map[string]any{
				PullPropertyRepoOwner: "mindersec",
				PullPropertyNumber:    int64(7036),
			},
			expectedErr: "error fetching pr name property",
		},
		{
			name: "missing pr number property",
			props: map[string]any{
				PullPropertyRepoOwner: "mindersec",
				PullPropertyRepoName:  "minder",
			},
			expectedErr: "error fetching pr id property",
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

func TestParsePrName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expectedOwner string
		expectedRepo  string
		expectedPRNum int64
		expectedErr   string
	}{
		{
			name:          "valid pr name",
			input:         "mindersec/minder/42",
			expectedOwner: "mindersec",
			expectedRepo:  "minder",
			expectedPRNum: 42,
		},
		{
			name:        "invalid parts count - too short",
			input:       "mindersec/minder",
			expectedErr: "invalid input format",
		},
		{
			name:        "invalid parts count - too long",
			input:       "mindersec/minder/42/extra",
			expectedErr: "invalid input format",
		},
		{
			name:        "invalid non-numeric pr number",
			input:       "mindersec/minder/forty-two",
			expectedErr: "invalid prNumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			owner, repo, prNum, err := parsePrName(tt.input)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedRepo, repo)
				assert.Equal(t, tt.expectedPRNum, prNum)
			}
		})
	}
}

func TestGetPrWrapperAttrsFromProps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		props         map[string]any
		expectedOwner string
		expectedRepo  string
		expectedPRNum int64
		expectedErr   string
	}{
		{
			name: "from individual owner, repo, and number properties",
			props: map[string]any{
				PullPropertyRepoOwner: "mindersec",
				PullPropertyRepoName:  "minder",
				PullPropertyNumber:    int64(100),
			},
			expectedOwner: "mindersec",
			expectedRepo:  "minder",
			expectedPRNum: 100,
		},
		{
			name: "from combined PropertyName",
			props: map[string]any{
				properties.PropertyName: "mindersec/minder/100",
			},
			expectedOwner: "mindersec",
			expectedRepo:  "minder",
			expectedPRNum: 100,
		},
		{
			name:        "missing required properties",
			props:       map[string]any{},
			expectedErr: "missing required properties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := properties.NewProperties(tt.props)

			owner, repo, prNum, err := getPrWrapperAttrsFromProps(p)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedRepo, repo)
				assert.Equal(t, tt.expectedPRNum, prNum)
			}
		})
	}
}

func TestPullRequestV1FromProperties(t *testing.T) {
	t.Parallel()

	props := map[string]any{
		properties.PullRequestUpstreamURL:    "https://github.com/mindersec/minder/pull/1",
		properties.PullRequestCommitSHA:      "abcdef123456",
		properties.PullRequestTargetBranch:   "feature",
		properties.PullRequestBaseBranch:     "main",
		PullPropertyNumber:                   int64(1),
		PullPropertyRepoOwner:                "mindersec",
		PullPropertyRepoName:                 "minder",
		PullPropertyAuthorID:                 int64(555),
		PullPropertyAction:                   "opened",
		properties.PullRequestBaseCloneURL:   "https://github.com/mindersec/minder.git",
		properties.PullRequestTargetCloneURL: "https://github.com/contributor/minder.git",
	}

	p := properties.NewProperties(props)

	prPb, err := PullRequestV1FromProperties(p)
	require.NoError(t, err)
	require.NotNil(t, prPb)

	assert.Equal(t, "https://github.com/mindersec/minder/pull/1", prPb.GetUrl())
	assert.Equal(t, "abcdef123456", prPb.GetCommitSha())
	assert.Equal(t, "feature", prPb.GetTargetRef())
	assert.Equal(t, "main", prPb.GetBaseRef())
	assert.Equal(t, int64(1), prPb.GetNumber())
	assert.Equal(t, "mindersec", prPb.GetRepoOwner())
	assert.Equal(t, "minder", prPb.GetRepoName())
	assert.Equal(t, int64(555), prPb.GetAuthorId())
	assert.Equal(t, "opened", prPb.GetAction())
	assert.Equal(t, "https://github.com/mindersec/minder.git", prPb.GetBaseCloneUrl())
	assert.Equal(t, "https://github.com/contributor/minder.git", prPb.GetTargetCloneUrl())
}
