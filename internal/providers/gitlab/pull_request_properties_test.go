// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"encoding/json"
	"maps"
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

func TestGetPropertiesForPullRequest_HTTPInteractions(t *testing.T) {
	t.Parallel()

	validLookupProps := map[string]any{
		properties.PropertyUpstreamID: "9001",
		PullRequestNumber:             "77",
		PullRequestProjectID:          "200",
	}

	t.Run("successful pull request properties fetch same project", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch r.URL.Path {
			case "/projects/200/merge_requests/77":
				mr := &gitlab.MergeRequest{
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
				}
				_ = json.NewEncoder(w).Encode(mr)
			case "/projects/200":
				proj := &gitlab.Project{
					ID:            200,
					Name:          "repo",
					HTTPURLToRepo: "https://gitlab.example.com/org/repo.git",
					Namespace: &gitlab.ProjectNamespace{
						Path: "org",
					},
				}
				_ = json.NewEncoder(w).Encode(proj)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		p := properties.NewProperties(validLookupProps)

		res, err := cli.getPropertiesForPullRequest(context.Background(), p)
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Equal(t, "9001", res.GetProperty(properties.PropertyUpstreamID).GetString())
		assert.Equal(t, "org/repo/77", res.GetProperty(properties.PropertyName).GetString())
		assert.Equal(t, "commit-sha-77", res.GetProperty(properties.PullRequestCommitSHA).GetString())
		assert.Equal(t, "main", res.GetProperty(properties.PullRequestBaseBranch).GetString())
		assert.Equal(t, "dev", res.GetProperty(properties.PullRequestTargetBranch).GetString())
		assert.Equal(t, int64(1234), res.GetProperty(PullRequestAuthor).GetInt64())
	})

	t.Run("successful pull request properties fetch from forked project", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch r.URL.Path {
			case "/projects/200/merge_requests/77":
				mr := &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						ID:              9001,
						IID:             77,
						SHA:             "fork-commit-sha",
						TargetBranch:    "main",
						SourceBranch:    "feature-fork",
						WebURL:          "https://gitlab.example.com/upstream/repo/-/merge_requests/77",
						SourceProjectID: 300, // Different source project (fork)
						Author: &gitlab.BasicUser{
							ID: 5566,
						},
					},
				}
				_ = json.NewEncoder(w).Encode(mr)
			case "/projects/200":
				proj := &gitlab.Project{
					ID:            200,
					Name:          "repo",
					HTTPURLToRepo: "https://gitlab.example.com/upstream/repo.git",
					Namespace: &gitlab.ProjectNamespace{
						Path: "upstream",
					},
				}
				_ = json.NewEncoder(w).Encode(proj)
			case "/projects/300":
				forkProj := &gitlab.Project{
					ID:            300,
					Name:          "repo-fork",
					HTTPURLToRepo: "https://gitlab.example.com/fork-user/repo-fork.git",
					Namespace: &gitlab.ProjectNamespace{
						Path: "fork-user",
					},
				}
				_ = json.NewEncoder(w).Encode(forkProj)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		p := properties.NewProperties(validLookupProps)

		res, err := cli.getPropertiesForPullRequest(context.Background(), p)
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Equal(t, "https://gitlab.example.com/upstream/repo.git", res.GetProperty(properties.PullRequestBaseCloneURL).GetString())
		assert.Equal(t, "https://gitlab.example.com/fork-user/repo-fork.git", res.GetProperty(properties.PullRequestTargetCloneURL).GetString())
	})

	t.Run("missing upstream ID in properties returns error", func(t *testing.T) {
		t.Parallel()

		cli := newTestGitlabClientWithServer(t, "https://example.com")
		props := maps.Clone(validLookupProps)
		delete(props, properties.PropertyUpstreamID)

		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(props))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream ID not found or invalid")
	})

	t.Run("missing merge request number in properties returns error", func(t *testing.T) {
		t.Parallel()

		cli := newTestGitlabClientWithServer(t, "https://example.com")
		props := maps.Clone(validLookupProps)
		delete(props, PullRequestNumber)

		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(props))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "merge request number not found or invalid")
	})

	t.Run("missing project ID in properties returns error", func(t *testing.T) {
		t.Parallel()

		cli := newTestGitlabClientWithServer(t, "https://example.com")
		props := maps.Clone(validLookupProps)
		delete(props, PullRequestProjectID)

		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(props))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID not found or invalid")
	})

	t.Run("merge request endpoint returns 404 returns ErrEntityNotFound", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(validLookupProps))
		require.ErrorIs(t, err, provifv1.ErrEntityNotFound)
	})

	t.Run("merge request endpoint returns 500 server error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get merge request")
	})

	t.Run("merge request endpoint returns malformed JSON", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{invalid-json"))
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")
	})

	t.Run("upstream ID mismatch with returned merge request ID returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			mr := &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					ID: 8888, // Mismatched from requested 9001
				},
			}
			_ = json.NewEncoder(w).Encode(mr)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "merge request ID mismatch: 8888 != 9001")
	})

	t.Run("project fetch failure returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/projects/200/merge_requests/77" {
				mr := &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						ID:  9001,
						IID: 77,
					},
				}
				_ = json.NewEncoder(w).Encode(mr)
				return
			}
			// Fail project call
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get project")
	})

	t.Run("target forked project fetch failure returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/projects/200/merge_requests/77":
				mr := &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						ID:              9001,
						IID:             77,
						SourceProjectID: 300,
					},
				}
				_ = json.NewEncoder(w).Encode(mr)
			case "/projects/200":
				proj := &gitlab.Project{
					ID:   200,
					Name: "repo",
					Namespace: &gitlab.ProjectNamespace{
						Path: "org",
					},
				}
				_ = json.NewEncoder(w).Encode(proj)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		cli := newTestGitlabClientWithServer(t, server.URL)
		_, err := cli.getPropertiesForPullRequest(context.Background(), properties.NewProperties(validLookupProps))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get target project")
	})
}
