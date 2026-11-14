// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/mindersec/minder/pkg/entities/properties"
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
