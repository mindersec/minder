// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"encoding/json"
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

func TestGetCommitBranchRefs(t *testing.T) {
	t.Parallel()

	var attempts int32
	mux := http.NewServeMux()

	// c0ffee: succeeds after transient 500 error
	mux.HandleFunc("/projects/123/repository/commits/c0ffee/refs", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "branch", r.URL.Query().Get("type"))
		w.Header().Set("Content-Type", "application/json")
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode([]*gitlablib.CommitRef{
			{Name: "main", Type: "branch"},
			{Name: "release-v1.0", Type: "branch"},
		})
	})

	// badcommit: fails after exceeding retries on persistent server error
	mux.HandleFunc("/projects/123/repository/commits/badcommit/refs", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	// badjson: returns decode error when response is malformed
	mux.HandleFunc("/projects/123/repository/commits/badjson/refs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid array}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cli := newTestGitlabClientWithServer(t, server.URL)

	tests := []struct {
		name            string
		commitSHA       string
		wantRefsCount   int
		wantFirstRef    string
		wantErrContains string
	}{
		{
			name:          "succeeds after transient 500 error",
			commitSHA:     "c0ffee",
			wantRefsCount: 2,
			wantFirstRef:  "main",
		},
		{
			name:            "fails after exceeding retries on server error",
			commitSHA:       "badcommit",
			wantErrContains: "failed to get commit refs",
		},
		{
			name:            "returns decode error when response is malformed",
			commitSHA:       "badjson",
			wantErrContains: "failed to decode response for commit refs",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			refs, err := cli.getCommitBranchRefs(context.Background(), "123", tc.commitSHA)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrContains)
				return
			}

			require.NoError(t, err)
			require.Len(t, refs, tc.wantRefsCount)
			assert.Equal(t, tc.wantFirstRef, refs[0].Name)
		})
	}
}

func TestGetPropertiesForRelease_HTTPInteractions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	// 100: successful release fetch
	mux.HandleFunc("/projects/100/releases/v2.5.0", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlablib.Release{
			TagName: "v2.5.0",
			Name:    "Release v2.5.0",
			Commit: gitlablib.Commit{
				ID: "sha1234567890abcdef",
			},
		})
	})
	mux.HandleFunc("/projects/100", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlablib.Project{
			ID:   100,
			Name: "service-repo",
			Namespace: &gitlablib.ProjectNamespace{
				Path: "org-group",
			},
		})
	})
	mux.HandleFunc("/projects/100/repository/commits/sha1234567890abcdef/refs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]*gitlablib.CommitRef{
			{Name: "main", Type: "branch"},
		})
	})

	// 404: release endpoint returns 404
	mux.HandleFunc("/projects/404/releases/v2.5.0", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// 500: release endpoint returns malformed JSON
	mux.HandleFunc("/projects/500/releases/v2.5.0", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid-json`))
	})

	// 600: project fetch failure (release succeeds, project fails with 500)
	mux.HandleFunc("/projects/600/releases/v2.5.0", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlablib.Release{
			TagName: "v2.5.0",
			Commit:  gitlablib.Commit{ID: "abc"},
		})
	})
	mux.HandleFunc("/projects/600", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	// 700: commit refs fetch failure (release & project succeed, commit refs fail with 500)
	mux.HandleFunc("/projects/700/releases/v2.5.0", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlablib.Release{
			TagName: "v2.5.0",
			Commit:  gitlablib.Commit{ID: "abc"},
		})
	})
	mux.HandleFunc("/projects/700", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlablib.Project{
			ID:        700,
			Name:      "repo",
			Namespace: &gitlablib.ProjectNamespace{Path: "ns"},
		})
	})
	mux.HandleFunc("/projects/700/repository/commits/abc/refs", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	// 800: empty commit refs returned (release & project succeed, empty refs array)
	mux.HandleFunc("/projects/800/releases/v2.5.0", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlablib.Release{
			TagName: "v2.5.0",
			Commit:  gitlablib.Commit{ID: "sha123"},
		})
	})
	mux.HandleFunc("/projects/800", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlablib.Project{
			ID:        800,
			Name:      "repo",
			Namespace: &gitlablib.ProjectNamespace{Path: "ns"},
		})
	})
	mux.HandleFunc("/projects/800/repository/commits/sha123/refs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]*gitlablib.CommitRef{})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cli := newTestGitlabClientWithServer(t, server.URL)

	tests := []struct {
		name            string
		props           map[string]any
		expectedProps   map[string]any
		wantErrContains string
	}{
		{
			name: "successful release properties fetch",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "100",
				ReleasePropertyTag:            "v2.5.0",
			},
			expectedProps: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "100",
				ReleasePropertyTag:            "v2.5.0",
				ReleasePropertyBranch:         "main",
				RepoPropertyNamespace:         "org-group",
				RepoPropertyProjectName:       "service-repo",
			},
		},
		{
			name: "missing upstream ID in input properties returns error",
			props: map[string]any{
				ReleasePropertyProjectID: "100",
				ReleasePropertyTag:       "v2.5.0",
			},
			wantErrContains: "upstream ID not found or invalid",
		},
		{
			name: "missing project ID in input properties returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyTag:            "v2.5.0",
			},
			wantErrContains: "project ID not found or invalid",
		},
		{
			name: "missing tag in input properties returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "100",
			},
			wantErrContains: "tag name not found or invalid",
		},
		{
			name: "release endpoint returns 404 error",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "404",
				ReleasePropertyTag:            "v2.5.0",
			},
			wantErrContains: "failed to get release using upstream ID",
		},
		{
			name: "release endpoint returns malformed JSON",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "500",
				ReleasePropertyTag:            "v2.5.0",
			},
			wantErrContains: "failed to decode response for release using upstream ID",
		},
		{
			name: "project fetch failure returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "600",
				ReleasePropertyTag:            "v2.5.0",
			},
			wantErrContains: "failed to get project",
		},
		{
			name: "commit refs fetch failure returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "700",
				ReleasePropertyTag:            "v2.5.0",
			},
			wantErrContains: "failed to get commit refs",
		},
		{
			name: "empty commit refs returned returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "999",
				ReleasePropertyProjectID:      "800",
				ReleasePropertyTag:            "v2.5.0",
			},
			wantErrContains: "no commit refs found for release",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := cli.getPropertiesForRelease(context.Background(), properties.NewProperties(tc.props))
			if tc.wantErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			for k, v := range tc.expectedProps {
				prop := res.GetProperty(k)
				require.NotNil(t, prop, "expected property %s to be present", k)
				switch exp := v.(type) {
				case string:
					assert.Equal(t, exp, prop.GetString(), "mismatch for property %s", k)
				case int64:
					assert.Equal(t, exp, prop.GetInt64(), "mismatch for property %s", k)
				default:
					assert.Equal(t, exp, prop.RawValue(), "mismatch for property %s", k)
				}
			}
		})
	}
}
