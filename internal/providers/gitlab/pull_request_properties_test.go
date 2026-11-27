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

	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestFormatPullRequestUpstreamID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "456", FormatPullRequestUpstreamID(456))
	assert.Equal(t, "0", FormatPullRequestUpstreamID(0))
}

func TestFormatPullRequestName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "group/project/12", formatPullRequestName("group", "project", "12"))
}

func TestGitlabMergeRequestToProperties(t *testing.T) {
	t.Parallel()

	t.Run("valid merge request without fork", func(t *testing.T) {
		t.Parallel()

		rawMR := `{
			"id": 101,
			"iid": 5,
			"sha": "abcdef1234567890",
			"target_branch": "main",
			"source_branch": "feature-branch",
			"web_url": "https://gitlab.com/mindersec/minder/-/merge_requests/5",
			"author": {
				"id": 777
			}
		}`
		mr := &gitlab.MergeRequest{}
		err := json.Unmarshal([]byte(rawMR), mr)
		require.NoError(t, err)

		rawProj := `{
			"id": 42,
			"name": "minder",
			"namespace": {
				"path": "mindersec"
			},
			"http_url_to_repo": "https://gitlab.com/mindersec/minder.git"
		}`
		proj := &gitlab.Project{}
		err = json.Unmarshal([]byte(rawProj), proj)
		require.NoError(t, err)

		props, err := gitlabMergeRequestToProperties(mr, proj, proj)
		require.NoError(t, err)
		require.NotNil(t, props)

		assert.Equal(t, "101", props.GetProperty(properties.PropertyUpstreamID).GetString())
		assert.Equal(t, "mindersec/minder/5", props.GetProperty(properties.PropertyName).GetString())
		assert.Equal(t, "abcdef1234567890", props.GetProperty(properties.PullRequestCommitSHA).GetString())
		assert.Equal(t, "https://gitlab.com/mindersec/minder.git", props.GetProperty(properties.PullRequestBaseCloneURL).GetString())
		assert.Equal(t, "main", props.GetProperty(properties.PullRequestBaseBranch).GetString())
		assert.Equal(t, "feature-branch", props.GetProperty(properties.PullRequestTargetBranch).GetString())
		assert.Equal(t, "https://gitlab.com/mindersec/minder/-/merge_requests/5", props.GetProperty(properties.PullRequestUpstreamURL).GetString())
		assert.Equal(t, "mindersec", props.GetProperty(RepoPropertyNamespace).GetString())
		assert.Equal(t, "minder", props.GetProperty(RepoPropertyProjectName).GetString())
		assert.Equal(t, "5", props.GetProperty(PullRequestNumber).GetString())
		assert.Equal(t, "42", props.GetProperty(PullRequestProjectID).GetString())
		assert.Equal(t, int64(777), props.GetProperty(PullRequestAuthor).GetInt64())
	})

	t.Run("missing namespace on project returns error", func(t *testing.T) {
		t.Parallel()

		rawMR := `{"id": 101, "iid": 5}`
		mr := &gitlab.MergeRequest{}
		err := json.Unmarshal([]byte(rawMR), mr)
		require.NoError(t, err)

		rawProj := `{"id": 42, "name": "orphan"}`
		proj := &gitlab.Project{}
		err = json.Unmarshal([]byte(rawProj), proj)
		require.NoError(t, err)

		props, err := gitlabMergeRequestToProperties(mr, proj, proj)
		require.Error(t, err)
		assert.Nil(t, props)
		assert.Contains(t, err.Error(), "failed to get namespace")
	})
}

func TestGetPullRequestNameFromProperties(t *testing.T) {
	t.Parallel()

	t.Run("valid properties", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyNamespace:   "org",
			RepoPropertyProjectName: "repo",
			PullRequestNumber:       "99",
		})

		name, err := getPullRequestNameFromProperties(p)
		require.NoError(t, err)
		assert.Equal(t, "org/repo/99", name)
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyProjectName: "repo",
			PullRequestNumber:       "99",
		})

		_, err := getPullRequestNameFromProperties(p)
		require.Error(t, err)
	})

	t.Run("missing project name returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyNamespace: "org",
			PullRequestNumber:     "99",
		})

		_, err := getPullRequestNameFromProperties(p)
		require.Error(t, err)
	})

	t.Run("missing pr number returns error", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(map[string]any{
			RepoPropertyNamespace:   "org",
			RepoPropertyProjectName: "repo",
		})

		_, err := getPullRequestNameFromProperties(p)
		require.Error(t, err)
	})
}

func TestPullRequestV1FromProperties(t *testing.T) {
	t.Parallel()

	validProps := map[string]any{
		properties.PropertyUpstreamID:           "999",
		PullRequestNumber:                       int64(42),
		RepoPropertyNamespace:                   "mindersec",
		RepoPropertyProjectName:                 "minder",
		properties.PullRequestCommitSHA:         "sha123",
		properties.PullRequestUpstreamURL:       "https://gitlab.com/mindersec/minder/-/merge_requests/42",
		PullRequestAuthor:                       int64(888),
		properties.PullRequestBaseCloneURL:      "https://gitlab.com/mindersec/minder.git",
		properties.PullRequestTargetCloneURL:    "https://gitlab.com/contributor/minder.git",
		properties.PullRequestBaseDefaultBranch: "main",
		properties.PullRequestTargetBranch:      "patch-1",
	}

	t.Run("successful conversion", func(t *testing.T) {
		t.Parallel()

		p := properties.NewProperties(validProps)
		prPb, err := pullRequestV1FromProperties(p)
		require.NoError(t, err)
		require.NotNil(t, prPb)

		assert.Equal(t, int64(42), prPb.GetNumber())
		assert.Equal(t, "mindersec", prPb.GetRepoOwner())
		assert.Equal(t, "minder", prPb.GetRepoName())
		assert.Equal(t, "sha123", prPb.GetCommitSha())
		assert.Equal(t, int64(888), prPb.GetAuthorId())
		assert.Equal(t, "https://gitlab.com/mindersec/minder/-/merge_requests/42", prPb.GetUrl())
		assert.Equal(t, "https://gitlab.com/mindersec/minder.git", prPb.GetBaseCloneUrl())
		assert.Equal(t, "https://gitlab.com/contributor/minder.git", prPb.GetTargetCloneUrl())
		assert.Equal(t, "main", prPb.GetBaseRef())
		assert.Equal(t, "patch-1", prPb.GetTargetRef())
	})

	t.Run("missing upstream ID returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			PullRequestNumber: int64(42),
		}
		p := properties.NewProperties(badProps)
		_, err := pullRequestV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get upstream ID")
	})

	t.Run("zero or missing number returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "999",
			PullRequestNumber:             int64(0),
		}
		p := properties.NewProperties(badProps)
		_, err := pullRequestV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get merge request number")
	})

	t.Run("missing namespace returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "999",
			PullRequestNumber:             int64(42),
		}
		p := properties.NewProperties(badProps)
		_, err := pullRequestV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get namespace")
	})

	t.Run("missing project name returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "999",
			PullRequestNumber:             int64(42),
			RepoPropertyNamespace:         "mindersec",
		}
		p := properties.NewProperties(badProps)
		_, err := pullRequestV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get project name")
	})

	t.Run("missing commit SHA returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID: "999",
			PullRequestNumber:             int64(42),
			RepoPropertyNamespace:         "mindersec",
			RepoPropertyProjectName:       "minder",
		}
		p := properties.NewProperties(badProps)
		_, err := pullRequestV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get commit SHA")
	})

	t.Run("missing merge request URL returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID:   "999",
			PullRequestNumber:               int64(42),
			RepoPropertyNamespace:           "mindersec",
			RepoPropertyProjectName:         "minder",
			properties.PullRequestCommitSHA: "sha123",
		}
		p := properties.NewProperties(badProps)
		_, err := pullRequestV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get merge request URL")
	})

	t.Run("missing author ID returns error", func(t *testing.T) {
		t.Parallel()

		badProps := map[string]any{
			properties.PropertyUpstreamID:     "999",
			PullRequestNumber:                 int64(42),
			RepoPropertyNamespace:             "mindersec",
			RepoPropertyProjectName:           "minder",
			properties.PullRequestCommitSHA:   "sha123",
			properties.PullRequestUpstreamURL: "https://gitlab.com",
		}
		p := properties.NewProperties(badProps)
		_, err := pullRequestV1FromProperties(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get author ID")
	})
}

func TestGetPropertiesForPullRequest(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	// Project metadata endpoints
	mux.HandleFunc("/projects/200", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlab.Project{
			ID:            200,
			Name:          "repo",
			HTTPURLToRepo: "https://gitlab.example.com/org/repo.git",
			Namespace: &gitlab.ProjectNamespace{
				Path: "org",
			},
		})
	})

	mux.HandleFunc("/projects/300", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlab.Project{
			ID:            300,
			Name:          "repo-fork",
			HTTPURLToRepo: "https://gitlab.example.com/fork-user/repo-fork.git",
			Namespace: &gitlab.ProjectNamespace{
				Path: "fork-user",
			},
		})
	})

	mux.HandleFunc("/projects/999", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	// MR endpoints
	// MR 77: standard same-project MR
	mux.HandleFunc("/projects/200/merge_requests/77", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID:              9001,
				IID:             77,
				SHA:             "commit-sha-77",
				TargetBranch:    "main",
				SourceBranch:    "dev",
				WebURL:          "https://gitlab.example.com/org/repo/-/merge_requests/77",
				SourceProjectID: 200,
				Author: &gitlab.BasicUser{
					ID: 1234,
				},
			},
		})
	})

	// MR 78: forked project MR
	mux.HandleFunc("/projects/200/merge_requests/78", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID:              9002,
				IID:             78,
				SHA:             "fork-commit-sha",
				TargetBranch:    "main",
				SourceBranch:    "feature-fork",
				WebURL:          "https://gitlab.example.com/upstream/repo/-/merge_requests/78",
				SourceProjectID: 300,
				Author: &gitlab.BasicUser{
					ID: 5566,
				},
			},
		})
	})

	// MR 79: source project fetch failure (source project 999 fails with 500)
	mux.HandleFunc("/projects/200/merge_requests/79", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID:              9003,
				IID:             79,
				SourceProjectID: 999,
			},
		})
	})

	// MR 88: ID mismatch (returns ID 8888 instead of requested upstream ID)
	mux.HandleFunc("/projects/200/merge_requests/88", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID: 8888,
			},
		})
	})

	// MR 404: not found
	mux.HandleFunc("/projects/200/merge_requests/404", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// MR 500: internal server error
	mux.HandleFunc("/projects/200/merge_requests/500", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	// MR 999: malformed JSON
	mux.HandleFunc("/projects/200/merge_requests/999", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{invalid-json"))
	})

	// Project 201 fails project fetch
	mux.HandleFunc("/projects/201/merge_requests/77", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID:  9001,
				IID: 77,
			},
		})
	})
	mux.HandleFunc("/projects/201", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cli := newTestGitlabClientWithServer(t, server.URL)

	testCases := []struct {
		name            string
		props           map[string]any
		expectedProps   map[string]any
		wantErrContains string
		wantErrIs       error
	}{
		{
			name: "successful pull request properties fetch same project",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestNumber:             "77",
				PullRequestProjectID:          "200",
			},
			expectedProps: map[string]any{
				properties.PropertyUpstreamID:      "9001",
				properties.PropertyName:            "org/repo/77",
				properties.PullRequestCommitSHA:    "commit-sha-77",
				properties.PullRequestBaseBranch:   "main",
				properties.PullRequestTargetBranch: "dev",
				PullRequestAuthor:                  int64(1234),
			},
		},
		{
			name: "successful pull request properties fetch from forked project",
			props: map[string]any{
				properties.PropertyUpstreamID: "9002",
				PullRequestNumber:             "78",
				PullRequestProjectID:          "200",
			},
			expectedProps: map[string]any{
				properties.PropertyUpstreamID:        "9002",
				properties.PropertyName:              "org/repo/78",
				properties.PullRequestBaseCloneURL:   "https://gitlab.example.com/org/repo.git",
				properties.PullRequestTargetCloneURL: "https://gitlab.example.com/fork-user/repo-fork.git",
			},
		},
		{
			name: "missing upstream ID in properties returns error",
			props: map[string]any{
				PullRequestNumber:    "77",
				PullRequestProjectID: "200",
			},
			wantErrContains: "upstream ID not found or invalid",
		},
		{
			name: "missing merge request number in properties returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestProjectID:          "200",
			},
			wantErrContains: "merge request number not found or invalid",
		},
		{
			name: "missing project ID in properties returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestNumber:             "77",
			},
			wantErrContains: "project ID not found or invalid",
		},
		{
			name: "merge request endpoint returns 404 returns ErrEntityNotFound",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestNumber:             "404",
				PullRequestProjectID:          "200",
			},
			wantErrIs: provifv1.ErrEntityNotFound,
		},
		{
			name: "merge request endpoint returns 500 server error",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestNumber:             "500",
				PullRequestProjectID:          "200",
			},
			wantErrContains: "failed to get merge request",
		},
		{
			name: "merge request endpoint returns malformed JSON",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestNumber:             "999",
				PullRequestProjectID:          "200",
			},
			wantErrContains: "failed to decode response",
		},
		{
			name: "upstream ID mismatch with returned merge request ID returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestNumber:             "88",
				PullRequestProjectID:          "200",
			},
			wantErrContains: "merge request ID mismatch: 8888 != 9001",
		},
		{
			name: "project fetch failure returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "9001",
				PullRequestNumber:             "77",
				PullRequestProjectID:          "201",
			},
			wantErrContains: "failed to get project",
		},
		{
			name: "target forked project fetch failure returns error",
			props: map[string]any{
				properties.PropertyUpstreamID: "9003",
				PullRequestNumber:             "79",
				PullRequestProjectID:          "200",
			},
			wantErrContains: "failed to get target project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := properties.NewProperties(tc.props)
			res, err := cli.getPropertiesForPullRequest(context.Background(), p)

			if tc.wantErrIs != nil {
				require.ErrorIs(t, err, tc.wantErrIs)
				return
			}
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
