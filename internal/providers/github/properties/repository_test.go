// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

import (
	"context"
	"testing"

	go_github "github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestNewRepositoryFetcher(t *testing.T) {
	t.Parallel()

	fetcher := NewRepositoryFetcher()
	require.NotNil(t, fetcher)
	assert.Len(t, fetcher.propertyOrigins, 1)
	assert.Len(t, fetcher.propertyOrigins[0].keys, 13)
	assert.Equal(t, repoOperationalProperties, fetcher.OperationalProperties())

	// Verify key definitions
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyName)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, properties.PropertyUpstreamID)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, RepoPropertyId)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, RepoPropertyName)
	assert.Contains(t, fetcher.propertyOrigins[0].keys, RepoPropertyOwner)

	// Verify wrapper lookup
	wrapper := fetcher.WrapperForProperty(RepoPropertyName)
	assert.NotNil(t, wrapper)
	assert.Nil(t, fetcher.WrapperForProperty("non_existent_key"))

	allWrappers := fetcher.AllPropertyWrappers()
	assert.Len(t, allWrappers, 1)
}

func TestRepositoryFetcher_GetName(t *testing.T) {
	t.Parallel()

	fetcher := NewRepositoryFetcher()

	tests := []struct {
		name         string
		props        map[string]any
		expectedName string
		expectedErr  string
	}{
		{
			name: "valid name and owner",
			props: map[string]any{
				RepoPropertyName:  "minder",
				RepoPropertyOwner: "mindersec",
			},
			expectedName: "mindersec/minder",
		},
		{
			name: "missing required owner property",
			props: map[string]any{
				RepoPropertyName: "minder",
			},
			expectedErr: "missing required properties",
		},
		{
			name: "missing required name property",
			props: map[string]any{
				RepoPropertyOwner: "mindersec",
			},
			expectedErr: "missing required properties",
		},
		{
			name: "empty name property value",
			props: map[string]any{
				RepoPropertyName:  "",
				RepoPropertyOwner: "mindersec",
			},
			expectedErr: "missing required repo-name property value",
		},
		{
			name: "empty owner property value",
			props: map[string]any{
				RepoPropertyName:  "minder",
				RepoPropertyOwner: "",
			},
			expectedErr: "missing required repo-owner property value",
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

func TestGetNameOwnerFromProps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		props         map[string]any
		expectedName  string
		expectedOwner string
		expectedErr   string
	}{
		{
			name: "from individual owner and name properties",
			props: map[string]any{
				RepoPropertyName:  "minder",
				RepoPropertyOwner: "mindersec",
			},
			expectedName:  "minder",
			expectedOwner: "mindersec",
		},
		{
			name: "from combined PropertyName",
			props: map[string]any{
				properties.PropertyName: "mindersec/minder",
			},
			expectedName:  "minder",
			expectedOwner: "mindersec",
		},
		{
			name: "invalid combined PropertyName format",
			props: map[string]any{
				properties.PropertyName: "mindersec/minder/extra",
			},
			expectedErr: "invalid repo name",
		},
		{
			name:        "missing all name properties",
			props:       map[string]any{},
			expectedErr: "missing required properties, either repo-name and repo-owner or name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := properties.NewProperties(tt.props)

			name, owner, err := getNameOwnerFromProps(context.Background(), p)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedOwner, owner)
			}
		})
	}
}

func TestGitHubRepoToMap(t *testing.T) {
	t.Parallel()

	id := int64(123456)
	name := "minder"
	owner := "mindersec"
	private := true
	archived := false
	fork := false
	deployURL := "https://api.github.com/repos/mindersec/minder/deployments"
	cloneURL := "https://github.com/mindersec/minder.git"
	defaultBranch := "main"
	licenseSPDX := "Apache-2.0"
	language := "Go"

	ghRepo := &go_github.Repository{
		ID:             &id,
		Name:           &name,
		Owner:          &go_github.User{Login: &owner},
		Private:        &private,
		Archived:       &archived,
		Fork:           &fork,
		DeploymentsURL: &deployURL,
		CloneURL:       &cloneURL,
		DefaultBranch:  &defaultBranch,
		License:        &go_github.License{SPDXID: &licenseSPDX},
		Language:       &language,
	}

	propMap := GitHubRepoToMap(ghRepo)
	require.NotNil(t, propMap)

	assert.Equal(t, "123456", propMap[properties.PropertyUpstreamID])
	assert.Equal(t, "mindersec/minder", propMap[properties.PropertyName])
	assert.Equal(t, true, propMap[properties.RepoPropertyIsPrivate])
	assert.Equal(t, false, propMap[properties.RepoPropertyIsArchived])
	assert.Equal(t, false, propMap[properties.RepoPropertyIsFork])
	assert.Equal(t, int64(123456), propMap[RepoPropertyId])
	assert.Equal(t, "minder", propMap[RepoPropertyName])
	assert.Equal(t, "mindersec", propMap[RepoPropertyOwner])
	assert.Equal(t, deployURL, propMap[RepoPropertyDeployURL])
	assert.Equal(t, cloneURL, propMap[RepoPropertyCloneURL])
	assert.Equal(t, "main", propMap[RepoPropertyDefaultBranch])
	assert.Equal(t, "Apache-2.0", propMap[RepoPropertyLicense])
	assert.Equal(t, "Go", propMap[RepoPropertyPrimaryLanguage])
}

func TestRepoV1FromProperties(t *testing.T) {
	t.Parallel()

	validProps := map[string]any{
		RepoPropertyName:                 "minder",
		RepoPropertyOwner:                "mindersec",
		RepoPropertyId:                   int64(987654),
		properties.RepoPropertyIsPrivate: true,
		properties.RepoPropertyIsFork:    false,
		RepoPropertyHookId:               int64(111),
		RepoPropertyHookUrl:              "https://webhook.minder.dev",
		RepoPropertyDeployURL:            "https://deploy.url",
		RepoPropertyCloneURL:             "https://clone.url",
		RepoPropertyHookType:             "repository",
		RepoPropertyHookName:             "web",
		RepoPropertyHookUiid:             "uiid-123",
		RepoPropertyDefaultBranch:        "main",
		RepoPropertyLicense:              "Apache-2.0",
	}

	t.Run("successful conversion", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(validProps)

		repoPb, err := RepoV1FromProperties(p)
		require.NoError(t, err)
		require.NotNil(t, repoPb)

		assert.Equal(t, "minder", repoPb.GetName())
		assert.Equal(t, "mindersec", repoPb.GetOwner())
		assert.Equal(t, int64(987654), repoPb.GetRepoId())
		assert.True(t, repoPb.GetIsPrivate())
		assert.False(t, repoPb.GetIsFork())
		assert.Equal(t, int64(111), repoPb.GetHookId())
		assert.Equal(t, "https://webhook.minder.dev", repoPb.GetHookUrl())
		assert.Equal(t, "main", repoPb.GetDefaultBranch())
		assert.Equal(t, "Apache-2.0", repoPb.GetLicense())
	})

	t.Run("error missing name", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			RepoPropertyOwner: "mindersec",
			RepoPropertyId:    int64(987654),
		}
		p := properties.NewProperties(badProps)

		_, err := RepoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching name property")
	})

	t.Run("error missing owner", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			RepoPropertyName: "minder",
			RepoPropertyId:   int64(987654),
		}
		p := properties.NewProperties(badProps)

		_, err := RepoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching owner property")
	})

	t.Run("error missing repo_id", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			RepoPropertyName:  "minder",
			RepoPropertyOwner: "mindersec",
		}
		p := properties.NewProperties(badProps)

		_, err := RepoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching repo_id property")
	})

	t.Run("error missing is_private", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			RepoPropertyName:  "minder",
			RepoPropertyOwner: "mindersec",
			RepoPropertyId:    int64(987654),
		}
		p := properties.NewProperties(badProps)

		_, err := RepoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching is_private property")
	})

	t.Run("error missing is_fork", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			RepoPropertyName:                 "minder",
			RepoPropertyOwner:                "mindersec",
			RepoPropertyId:                   int64(987654),
			properties.RepoPropertyIsPrivate: false,
		}
		p := properties.NewProperties(badProps)

		_, err := RepoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching is_fork property")
	})
}
