// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlablib "gitlab.com/gitlab-org/api/client-go"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestFormatReleaseUpstreamID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "789", FormatReleaseUpstreamID(789))
	assert.Equal(t, "0", FormatReleaseUpstreamID(0))
}

func TestFormatReleaseName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "mindersec/minder/v1.0.0", formatReleaseName("mindersec", "minder", "v1.0.0"))
}

func TestGuessBranchFromCommitRefs(t *testing.T) {
	t.Parallel()

	t.Run("single ref returned directly", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "feature/foo"},
		}
		assert.Equal(t, "feature/foo", guessBranchFromCommitRefs(refs, "v1.0.0"))
	})

	t.Run("ref matching exact tag name", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "feature/foo"},
			{Name: "v1.2.3"},
			{Name: "feature/bar"},
		}
		assert.Equal(t, "v1.2.3", guessBranchFromCommitRefs(refs, "v1.2.3"))
	})

	t.Run("ref with release- prefix", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "some-feature"},
			{Name: "release-v2"},
			{Name: "dev"},
		}
		assert.Equal(t, "release-v2", guessBranchFromCommitRefs(refs, "v2.0.0"))
	})

	t.Run("ref with release/ prefix", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "some-feature"},
			{Name: "release/1.0"},
		}
		assert.Equal(t, "release/1.0", guessBranchFromCommitRefs(refs, "v1.0.1"))
	})

	t.Run("semver major match", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "random-branch"},
			{Name: "v3.1.0"},
		}
		assert.Equal(t, "v3.1.0", guessBranchFromCommitRefs(refs, "v3.0.0"))
	})

	t.Run("ref starts with tag", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "feature-x"},
			{Name: "rc1-hotfix"},
		}
		assert.Equal(t, "rc1-hotfix", guessBranchFromCommitRefs(refs, "rc1"))
	})

	t.Run("tag starts with ref", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "feature-x"},
			{Name: "build-42"},
		}
		assert.Equal(t, "build-42", guessBranchFromCommitRefs(refs, "build-42-final"))
	})

	t.Run("main or master fallback", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "feature-foo"},
			{Name: "main"},
		}
		assert.Equal(t, "main", guessBranchFromCommitRefs(refs, "custom-tag"))
	})

	t.Run("first ref fallback when no heuristic matches", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "alpha-build"},
			{Name: "beta-build"},
		}
		assert.Equal(t, "alpha-build", guessBranchFromCommitRefs(refs, "gamma-tag"))
	})
}

func TestGitlabReleaseToProperties(t *testing.T) {
	t.Parallel()

	proj := &gitlablib.Project{
		ID:   42,
		Name: "minder",
		Namespace: &gitlablib.ProjectNamespace{
			Path: "mindersec",
		},
	}

	props := gitlabReleaseToProperties("rel-123", "v1.0.0", proj, "main")
	require.NotNil(t, props)

	assert.Equal(t, "rel-123", props.GetProperty(properties.PropertyUpstreamID).GetString())
	assert.Equal(t, "42", props.GetProperty(ReleasePropertyProjectID).GetString())
	assert.Equal(t, "v1.0.0", props.GetProperty(ReleasePropertyTag).GetString())
	assert.Equal(t, "main", props.GetProperty(ReleasePropertyBranch).GetString())
	assert.Equal(t, "mindersec", props.GetProperty(RepoPropertyNamespace).GetString())
	assert.Equal(t, "minder", props.GetProperty(RepoPropertyProjectName).GetString())
}

func TestGetReleaseNameFromProperties(t *testing.T) {
	t.Parallel()

	t.Run("valid properties", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			ReleasePropertyTag:      "v1.0.0",
			RepoPropertyNamespace:   "mindersec",
			RepoPropertyProjectName: "minder",
		})

		name, err := getReleaseNameFromProperties(p)
		require.NoError(t, err)
		assert.Equal(t, "mindersec/minder/v1.0.0", name)
	})

	t.Run("missing tag returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyNamespace:   "mindersec",
			RepoPropertyProjectName: "minder",
		})

		_, err := getReleaseNameFromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch not found or invalid")
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			ReleasePropertyTag:      "v1.0.0",
			RepoPropertyProjectName: "minder",
		})

		_, err := getReleaseNameFromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "namespace not found or invalid")
	})

	t.Run("missing project name returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			ReleasePropertyTag:    "v1.0.0",
			RepoPropertyNamespace: "mindersec",
		})

		_, err := getReleaseNameFromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project name not found or invalid")
	})
}

func TestReleaseEntityV1FromProperties(t *testing.T) {
	t.Parallel()

	validProps := map[string]any{
		properties.PropertyUpstreamID: "rel-456",
		ReleasePropertyProjectID:      "42",
		ReleasePropertyBranch:         "main",
		ReleasePropertyTag:            "v2.0.0",
		RepoPropertyNamespace:         "mindersec",
		RepoPropertyProjectName:       "minder",
	}

	t.Run("successful conversion", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(validProps)
		entity, err := releaseEntityV1FromProperties(p)
		require.NoError(t, err)
		require.NotNil(t, entity)

		assert.Equal(t, minderv1.Entity_ENTITY_RELEASE, entity.GetType())
		assert.Equal(t, "mindersec/minder/v2.0.0", entity.GetName())
	})

	t.Run("missing upstream ID returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			ReleasePropertyProjectID: "42",
		}
		p := properties.NewProperties(badProps)
		_, err := releaseEntityV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream ID not found")
	})

	t.Run("missing project ID returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "rel-456",
		}
		p := properties.NewProperties(badProps)
		_, err := releaseEntityV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID not found")
	})

	t.Run("missing branch returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "rel-456",
			ReleasePropertyProjectID:      "42",
		}
		p := properties.NewProperties(badProps)
		_, err := releaseEntityV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch not found")
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "rel-456",
			ReleasePropertyProjectID:      "42",
			ReleasePropertyBranch:         "main",
		}
		p := properties.NewProperties(badProps)
		_, err := releaseEntityV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "namespace not found")
	})

	t.Run("missing project name returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "rel-456",
			ReleasePropertyProjectID:      "42",
			ReleasePropertyBranch:         "main",
			RepoPropertyNamespace:         "mindersec",
		}
		p := properties.NewProperties(badProps)
		_, err := releaseEntityV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project name not found")
	})
}

func TestGetCommitBranchRefs_RetryAndSuccess(t *testing.T) {
	t.Parallel()

	t.Run("succeeds after transient 500 error", func(t *testing.T) {
		t.Parallel()

		var attempts int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			count := atomic.AddInt32(&attempts, 1)

			assert.Contains(t, r.URL.Path, "commits/c0ffee/refs")
			assert.Equal(t, "branch", r.URL.Query().Get("type"))

			if count == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			refs := []*gitlablib.CommitRef{
				{Name: "main", Type: "branch"},
				{Name: "release-v1.0", Type: "branch"},
			}
			_ = json.NewEncoder(w).Encode(refs)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		refs, err := cli.getCommitBranchRefs(context.Background(), "123", "c0ffee")
		require.NoError(t, err)
		require.Len(t, refs, 2)
		assert.Equal(t, "main", refs[0].Name)
		assert.Equal(t, "release-v1.0", refs[1].Name)
		assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
	})

	t.Run("fails after exceeding retries on server error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getCommitBranchRefs(context.Background(), "123", "badcommit")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get commit refs")
	})

	t.Run("returns decode error when response is malformed", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not valid array}`))
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getCommitBranchRefs(context.Background(), "123", "badjson")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response for commit refs")
	})
}

func TestGetPropertiesForRelease_HTTPInteractions(t *testing.T) {
	t.Parallel()

	validLookupProps := map[string]any{
		properties.PropertyUpstreamID: "999",
		ReleasePropertyProjectID:      "100",
		ReleasePropertyTag:            "v2.5.0",
	}

	t.Run("successful release properties fetch", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch r.URL.Path {
			case "/projects/100/releases/v2.5.0":
				release := &gitlablib.Release{
					TagName: "v2.5.0",
					Name:    "Release v2.5.0",
					Commit: gitlablib.Commit{
						ID: "sha1234567890abcdef",
					},
				}
				_ = json.NewEncoder(w).Encode(release)
			case "/projects/100":
				proj := &gitlablib.Project{
					ID:   100,
					Name: "service-repo",
					Namespace: &gitlablib.ProjectNamespace{
						Path: "org-group",
					},
				}
				_ = json.NewEncoder(w).Encode(proj)
			case "/projects/100/repository/commits/sha1234567890abcdef/refs":
				refs := []*gitlablib.CommitRef{
					{Name: "main", Type: "branch"},
				}
				_ = json.NewEncoder(w).Encode(refs)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		res, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(validLookupProps))
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Equal(t, "999", res.GetProperty(properties.PropertyUpstreamID).GetString())
		assert.Equal(t, "100", res.GetProperty(ReleasePropertyProjectID).GetString())
		assert.Equal(t, "v2.5.0", res.GetProperty(ReleasePropertyTag).GetString())
		assert.Equal(t, "main", res.GetProperty(ReleasePropertyBranch).GetString())
		assert.Equal(t, "org-group", res.GetProperty(RepoPropertyNamespace).GetString())
		assert.Equal(t, "service-repo", res.GetProperty(RepoPropertyProjectName).GetString())
	})

	t.Run("missing upstream ID in input properties returns error", func(t *testing.T) {
		t.Parallel()

		cli := newTestGitlabClientWithServer(t, "https://example.com")
		props := maps.Clone(validLookupProps)
		delete(props, properties.PropertyUpstreamID)

		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(props))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream ID not found or invalid")
	})

	t.Run("missing project ID in input properties returns error", func(t *testing.T) {
		t.Parallel()

		cli := newTestGitlabClientWithServer(t, "https://example.com")
		props := maps.Clone(validLookupProps)
		delete(props, ReleasePropertyProjectID)

		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(props))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID not found or invalid")
	})

	t.Run("missing tag in input properties returns error", func(t *testing.T) {
		t.Parallel()

		cli := newTestGitlabClientWithServer(t, "https://example.com")
		props := maps.Clone(validLookupProps)
		delete(props, ReleasePropertyTag)

		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(props))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag name not found or invalid")
	})

	t.Run("release endpoint returns 404 error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get release using upstream ID")
	})

	t.Run("release endpoint returns malformed JSON", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{invalid-json`))
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response for release using upstream ID")
	})

	t.Run("project fetch failure returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/projects/100/releases/v2.5.0" {
				release := &gitlablib.Release{
					TagName: "v2.5.0",
					Commit:  gitlablib.Commit{ID: "abc"},
				}
				_ = json.NewEncoder(w).Encode(release)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get project")
	})

	t.Run("commit refs fetch failure returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/projects/100/releases/v2.5.0":
				release := &gitlablib.Release{
					TagName: "v2.5.0",
					Commit:  gitlablib.Commit{ID: "abc"},
				}
				_ = json.NewEncoder(w).Encode(release)
			case "/projects/100":
				proj := &gitlablib.Project{
					ID:        100,
					Name:      "repo",
					Namespace: &gitlablib.ProjectNamespace{Path: "ns"},
				}
				_ = json.NewEncoder(w).Encode(proj)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get commit refs")
	})

	t.Run("empty commit refs returned returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/projects/100/releases/v2.5.0":
				release := &gitlablib.Release{
					TagName: "v2.5.0",
					Commit:  gitlablib.Commit{ID: "sha123"},
				}
				_ = json.NewEncoder(w).Encode(release)
			case "/projects/100":
				proj := &gitlablib.Project{
					ID:        100,
					Name:      "repo",
					Namespace: &gitlablib.ProjectNamespace{Path: "ns"},
				}
				_ = json.NewEncoder(w).Encode(proj)
			case "/projects/100/repository/commits/sha123/refs":
				refs := []*gitlablib.CommitRef{}
				_ = json.NewEncoder(w).Encode(refs)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no commit refs found for release")
	})
}
