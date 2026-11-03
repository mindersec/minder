// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/mindersec/minder/internal/providers/credentials"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestFormatRepositoryUpstreamID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "12345", FormatRepositoryUpstreamID(12345))
	assert.Equal(t, "0", FormatRepositoryUpstreamID(0))
}

func TestFormatRepoName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "group/project", formatRepoName("group", "project"))
	assert.Equal(t, "subgroup/group/project", formatRepoName("subgroup/group", "project"))
}

func TestGitlabProjectToProperties(t *testing.T) {
	t.Parallel()

	t.Run("valid project with license and namespace", func(t *testing.T) {
		t.Parallel()

		proj := &gitlab.Project{
			ID:         42,
			Name:       "minder",
			Visibility: gitlab.PrivateVisibility,
			Archived:   false,
			Namespace: &gitlab.ProjectNamespace{
				Path: "mindersec",
			},
			DefaultBranch: "main",
			License: &gitlab.ProjectLicense{
				Name: "Apache-2.0",
			},
			HTTPURLToRepo: "https://gitlab.com/mindersec/minder.git",
		}

		props, err := gitlabProjectToProperties(proj)
		require.NoError(t, err)
		require.NotNil(t, props)

		assert.Equal(t, "42", props.GetProperty(properties.PropertyUpstreamID).GetString())
		assert.Equal(t, "mindersec/minder", props.GetProperty(properties.PropertyName).GetString())
		assert.True(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool())
		assert.False(t, props.GetProperty(properties.RepoPropertyIsArchived).GetBool())
		assert.False(t, props.GetProperty(properties.RepoPropertyIsFork).GetBool())
		assert.Equal(t, "main", props.GetProperty(RepoPropertyDefaultBranch).GetString())
		assert.Equal(t, "mindersec", props.GetProperty(RepoPropertyNamespace).GetString())
		assert.Equal(t, "minder", props.GetProperty(RepoPropertyProjectName).GetString())
		assert.Equal(t, "Apache-2.0", props.GetProperty(RepoPropertyLicense).GetString())
		assert.Equal(t, "https://gitlab.com/mindersec/minder.git", props.GetProperty(RepoPropertyCloneURL).GetString())
	})

	t.Run("valid project forked without license", func(t *testing.T) {
		t.Parallel()

		proj := &gitlab.Project{
			ID:                100,
			Name:              "forked-proj",
			Visibility:        gitlab.PublicVisibility,
			Archived:          true,
			ForkedFromProject: &gitlab.ForkParent{ID: 1},
			Namespace: &gitlab.ProjectNamespace{
				Path: "user",
			},
			DefaultBranch: "develop",
			HTTPURLToRepo: "https://gitlab.com/user/forked-proj.git",
		}

		props, err := gitlabProjectToProperties(proj)
		require.NoError(t, err)
		require.NotNil(t, props)

		assert.Equal(t, "100", props.GetProperty(properties.PropertyUpstreamID).GetString())
		assert.Equal(t, "user/forked-proj", props.GetProperty(properties.PropertyName).GetString())
		assert.False(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool())
		assert.True(t, props.GetProperty(properties.RepoPropertyIsArchived).GetBool())
		assert.True(t, props.GetProperty(properties.RepoPropertyIsFork).GetBool())
		assert.Equal(t, "", props.GetProperty(RepoPropertyLicense).GetString())
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		t.Parallel()

		proj := &gitlab.Project{
			ID:   50,
			Name: "orphan-proj",
		}

		props, err := gitlabProjectToProperties(proj)
		require.Error(t, err)
		assert.Nil(t, props)
		assert.Contains(t, err.Error(), "has no namespace")
	})
}

func TestGetRepoNameFromProperties(t *testing.T) {
	t.Parallel()

	t.Run("valid namespace and project name", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyNamespace:   "gitlab-org",
			RepoPropertyProjectName: "gitlab",
		})

		name, err := getRepoNameFromProperties(p)
		require.NoError(t, err)
		assert.Equal(t, "gitlab-org/gitlab", name)
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyProjectName: "gitlab",
		})

		_, err := getRepoNameFromProperties(p)
		require.Error(t, err)
	})

	t.Run("missing project name returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyNamespace: "gitlab-org",
		})

		_, err := getRepoNameFromProperties(p)
		require.Error(t, err)
	})
}

func TestRepoV1FromProperties(t *testing.T) {
	t.Parallel()

	validProps := map[string]any{
		properties.PropertyUpstreamID:    "123456",
		RepoPropertyProjectName:          "minder",
		RepoPropertyNamespace:            "mindersec",
		properties.RepoPropertyIsPrivate: true,
		properties.RepoPropertyIsFork:    false,
		RepoPropertyCloneURL:             "https://gitlab.com/mindersec/minder.git",
		RepoPropertyDefaultBranch:        "main",
		RepoPropertyLicense:              "Apache-2.0",
	}

	t.Run("successful conversion", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(validProps)
		repoPb, err := repoV1FromProperties(p)
		require.NoError(t, err)
		require.NotNil(t, repoPb)

		assert.Equal(t, "minder", repoPb.GetName())
		assert.Equal(t, "mindersec", repoPb.GetOwner())
		assert.Equal(t, int64(123456), repoPb.GetRepoId())
		assert.True(t, repoPb.GetIsPrivate())
		assert.False(t, repoPb.GetIsFork())
		assert.Equal(t, "https://gitlab.com/mindersec/minder.git", repoPb.GetCloneUrl())
		assert.Equal(t, "main", repoPb.GetDefaultBranch())
		assert.Equal(t, "Apache-2.0", repoPb.GetLicense())
	})

	t.Run("missing upstream ID returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			RepoPropertyProjectName: "minder",
			RepoPropertyNamespace:   "mindersec",
		}
		p := properties.NewProperties(badProps)
		_, err := repoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching upstream ID property")
	})

	t.Run("invalid non-numeric upstream ID returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "not-a-number",
			RepoPropertyProjectName:       "minder",
			RepoPropertyNamespace:         "mindersec",
		}
		p := properties.NewProperties(badProps)
		_, err := repoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error converting upstream ID to int64")
	})

	t.Run("missing project name returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "123",
			RepoPropertyNamespace:         "mindersec",
		}
		p := properties.NewProperties(badProps)
		_, err := repoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching project property")
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "123",
			RepoPropertyProjectName:       "minder",
		}
		p := properties.NewProperties(badProps)
		_, err := repoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching namespace property")
	})

	t.Run("missing is_private returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "123",
			RepoPropertyProjectName:       "minder",
			RepoPropertyNamespace:         "mindersec",
		}
		p := properties.NewProperties(badProps)
		_, err := repoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching is_private property")
	})

	t.Run("missing is_fork returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID:    "123",
			RepoPropertyProjectName:          "minder",
			RepoPropertyNamespace:            "mindersec",
			properties.RepoPropertyIsPrivate: true,
		}
		p := properties.NewProperties(badProps)
		_, err := repoV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching is_fork property")
	})
}

func TestGetPropertiesForRepo_HTTPInteractions(t *testing.T) {
	t.Parallel()

	t.Run("successful repo properties fetch from gitlab server", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "projects/12345")
			assert.Equal(t, "true", r.URL.Query().Get("license"))

			proj := &gitlab.Project{
				ID:            12345,
				Name:          "demo-app",
				DefaultBranch: "main",
				Visibility:    gitlab.PrivateVisibility,
				Archived:      false,
				HTTPURLToRepo: "https://gitlab.example.com/group/demo-app.git",
				Namespace: &gitlab.ProjectNamespace{
					Path: "group",
				},
				License: &gitlab.ProjectLicense{
					Name: "BSD-3-Clause",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(proj)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)

		inputProps := properties.NewProperties(map[string]any{
			properties.PropertyUpstreamID: "12345",
			"custom-existing-prop":        "keep-me",
		})

		mergedProps, err := cli.getPropertiesForRepo(context.Background(), inputProps)
		require.NoError(t, err)
		require.NotNil(t, mergedProps)

		assert.Equal(t, "12345", mergedProps.GetProperty(properties.PropertyUpstreamID).GetString())
		assert.Equal(t, "group/demo-app", mergedProps.GetProperty(properties.PropertyName).GetString())
		assert.Equal(t, "group", mergedProps.GetProperty(RepoPropertyNamespace).GetString())
		assert.Equal(t, "demo-app", mergedProps.GetProperty(RepoPropertyProjectName).GetString())
		assert.Equal(t, "BSD-3-Clause", mergedProps.GetProperty(RepoPropertyLicense).GetString())
		assert.True(t, mergedProps.GetProperty(properties.RepoPropertyIsPrivate).GetBool())
		assert.Equal(t, "keep-me", mergedProps.GetProperty("custom-existing-prop").GetString())
	})

	t.Run("missing upstream ID in getByProps returns error", func(t *testing.T) {
		t.Parallel()

		cli := newTestGitlabClientWithServer(t, "https://example.com")
		inputProps := properties.NewProperties(map[string]any{})

		_, err := cli.getPropertiesForRepo(context.Background(), inputProps)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream ID not found or invalid")
	})

	t.Run("project not found returns ErrEntityNotFound", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)

		inputProps := properties.NewProperties(map[string]any{
			properties.PropertyUpstreamID: "999999",
		})

		_, err := cli.getPropertiesForRepo(context.Background(), inputProps)
		require.ErrorIs(t, err, provifv1.ErrEntityNotFound)
	})

	t.Run("server returns 500 internal server error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)

		inputProps := properties.NewProperties(map[string]any{
			properties.PropertyUpstreamID: "123",
		})

		_, err := cli.getPropertiesForRepo(context.Background(), inputProps)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get projects")
	})

	t.Run("server returns malformed JSON body", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{invalid-json-body"))
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)

		inputProps := properties.NewProperties(map[string]any{
			properties.PropertyUpstreamID: "123",
		})

		_, err := cli.getPropertiesForRepo(context.Background(), inputProps)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")
	})

	t.Run("server returns project without namespace triggers conversion error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			proj := &gitlab.Project{
				ID:        555,
				Name:      "no-namespace",
				Namespace: nil,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(proj)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)

		inputProps := properties.NewProperties(map[string]any{
			properties.PropertyUpstreamID: "555",
		})

		_, err := cli.getPropertiesForRepo(context.Background(), inputProps)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to convert project to properties")
	})
}

func newTestGitlabClientWithServer(t *testing.T, serverURL string) *gitlabClient {
	t.Helper()

	cli, err := New(
		context.Background(),
		credentials.NewGitLabTokenCredential("glpat-test-secret-token"),
		&minderv1.GitLabProviderConfig{
			Endpoint: serverURL,
		},
		"https://example.com/webhook",
		"webhook-secret",
		server.GitConfig{},
	)
	require.NoError(t, err)
	return cli
}
