// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
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

	t.Run("tag prefix matching", func(t *testing.T) {
		t.Parallel()

		refs := []*gitlablib.CommitRef{
			{Name: "feature-branch"},
			{Name: "v1.0"},
		}
		assert.Equal(t, "v1.0", guessBranchFromCommitRefs(refs, "v1.0.5"))
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
