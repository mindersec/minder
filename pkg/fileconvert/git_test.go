// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/fileconvert"
)

func TestParseGitURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		rawURL      string
		expectedURL string
		expectedDir string
		expectErr   bool
	}{
		{
			name:        "valid url without subdir",
			rawURL:      "https://github.com/mindersec/minder-rules-and-profiles",
			expectedURL: "https://github.com/mindersec/minder-rules-and-profiles",
			expectedDir: "",
			expectErr:   false,
		},
		{
			name:        "valid url with subdir",
			rawURL:      "https://github.com/mindersec/minder-rules-and-profiles?security-baseline",
			expectedURL: "https://github.com/mindersec/minder-rules-and-profiles",
			expectedDir: "security-baseline",
			expectErr:   false,
		},
		{
			name:        "valid url with leading slash in query",
			rawURL:      "https://github.com/mindersec/minder-rules-and-profiles?/nested/dir/path",
			expectedURL: "https://github.com/mindersec/minder-rules-and-profiles",
			expectedDir: "nested/dir/path",
			expectErr:   false,
		},
		{
			name:        "valid url with trailing slash in query",
			rawURL:      "https://github.com/mindersec/minder-rules-and-profiles?nested/dir/path/",
			expectedURL: "https://github.com/mindersec/minder-rules-and-profiles",
			expectedDir: "nested/dir/path",
			expectErr:   false,
		},
		{
			name:        "valid url with empty query",
			rawURL:      "https://github.com/mindersec/minder-rules-and-profiles?",
			expectedURL: "https://github.com/mindersec/minder-rules-and-profiles",
			expectedDir: "",
			expectErr:   false,
		},
		{
			name:        "valid url with dot query",
			rawURL:      "https://github.com/mindersec/minder-rules-and-profiles?.",
			expectedURL: "https://github.com/mindersec/minder-rules-and-profiles",
			expectedDir: "",
			expectErr:   false,
		},
		{
			name:        "valid file url without host",
			rawURL:      "file:///path/to/repo?subdir",
			expectedURL: "file:///path/to/repo",
			expectedDir: "subdir",
			expectErr:   false,
		},
		{
			name:      "empty input url",
			rawURL:    "",
			expectErr: true,
		},
		{
			name:      "missing scheme or host",
			rawURL:    "github.com/mindersec/minder-rules-and-profiles",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repoURL, subDir, err := fileconvert.ParseGitURL(tc.rawURL)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedURL, repoURL)
				assert.Equal(t, tc.expectedDir, subDir)
			}
		})
	}
}

func TestResourcesFromGit(t *testing.T) {
	t.Parallel()

	profileYaml := `
---
version: v1
type: profile
name: test-profile
context:
  provider: github
`
	ruleTypeYaml := `
---
version: v1
type: rule-type
name: test-ruletype
context:
  provider: github
def:
  in_entity: artifact
  rule_schema: {}
  ingest:
    type: fake
  eval:
    type: other
`
	nonResourceYaml := `
foo: bar
baz: qux
`

	// Create a real temporary on-disk git repository to test ResourcesFromGit with a file:// URL
	tmpDir := t.TempDir()
	repo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "rules"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "other"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "rules", "profile.yaml"), []byte(profileYaml), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "rules", "rule.yaml"), []byte(ruleTypeYaml), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "rules", "ignored.txt"), []byte("not a resource"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other", "something.yaml"), []byte(nonResourceYaml), 0600))

	w, err := repo.Worktree()
	require.NoError(t, err)
	_, err = w.Add(".")
	require.NoError(t, err)
	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	repoURI := filepath.ToSlash(tmpDir)
	if !filepath.IsAbs(repoURI) {
		absDir, err := filepath.Abs(tmpDir)
		require.NoError(t, err)
		repoURI = filepath.ToSlash(absDir)
	}
	if !strings.HasPrefix(repoURI, "/") {
		repoURI = "/" + repoURI
	}
	fileGitURL := "file://" + repoURI

	t.Run("read git repo root", func(t *testing.T) {
		t.Parallel()
		resources, err := fileconvert.ResourcesFromGit(context.Background(), nil, fileGitURL)
		require.NoError(t, err)
		assert.Len(t, resources, 2)
	})

	t.Run("read git repo subdir", func(t *testing.T) {
		t.Parallel()
		resources, err := fileconvert.ResourcesFromGit(context.Background(), nil, fileGitURL+"?rules")
		require.NoError(t, err)
		assert.Len(t, resources, 2)

		var names []string
		for _, res := range resources {
			names = append(names, res.GetName())
		}
		assert.Contains(t, names, "test-profile")
		assert.Contains(t, names, "test-ruletype")
	})

	t.Run("skip non-resource files in subdir", func(t *testing.T) {
		t.Parallel()
		resources, err := fileconvert.ResourcesFromGit(context.Background(), func(_ string, _ ...any) {}, fileGitURL+"?other")
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("invalid git URL error", func(t *testing.T) {
		t.Parallel()
		_, err := fileconvert.ResourcesFromGit(context.Background(), nil, "invalid-url")
		require.Error(t, err)
	})
}
