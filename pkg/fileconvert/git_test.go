// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert_test

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

func TestResourcesFromFS(t *testing.T) {
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

	createTestFS := func(t *testing.T) billy.Filesystem {
		t.Helper()
		fs := memfs.New()
		require.NoError(t, fs.MkdirAll("rules", 0755))
		require.NoError(t, fs.MkdirAll("other", 0755))

		p, err := fs.Create("rules/profile.yaml")
		require.NoError(t, err)
		_, err = p.Write([]byte(profileYaml))
		require.NoError(t, err)
		p.Close()

		r, err := fs.Create("rules/rule.yaml")
		require.NoError(t, err)
		_, err = r.Write([]byte(ruleTypeYaml))
		require.NoError(t, err)
		r.Close()

		txt, err := fs.Create("rules/ignored.txt")
		require.NoError(t, err)
		_, err = txt.Write([]byte("not a resource"))
		require.NoError(t, err)
		txt.Close()

		nr, err := fs.Create("other/something.yaml")
		require.NoError(t, err)
		_, err = nr.Write([]byte(nonResourceYaml))
		require.NoError(t, err)
		nr.Close()

		return fs
	}

	t.Run("read whole fs root", func(t *testing.T) {
		t.Parallel()
		fs := createTestFS(t)
		resources, err := fileconvert.ResourcesFromFS(nil, fs, "")
		require.NoError(t, err)
		assert.Len(t, resources, 2)
	})

	t.Run("read subdirectory rules", func(t *testing.T) {
		t.Parallel()
		fs := createTestFS(t)
		resources, err := fileconvert.ResourcesFromFS(nil, fs, "rules")
		require.NoError(t, err)
		assert.Len(t, resources, 2)

		var names []string
		for _, res := range resources {
			names = append(names, res.GetName())
		}
		assert.Contains(t, names, "test-profile")
		assert.Contains(t, names, "test-ruletype")
	})

	t.Run("read single file directly", func(t *testing.T) {
		t.Parallel()
		fs := createTestFS(t)
		resources, err := fileconvert.ResourcesFromFS(nil, fs, "rules/profile.yaml")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		prof, ok := resources[0].(*minderv1.Profile)
		require.True(t, ok)
		assert.Equal(t, "test-profile", prof.GetName())
	})

	t.Run("read non-existent path", func(t *testing.T) {
		t.Parallel()
		fs := createTestFS(t)
		_, err := fileconvert.ResourcesFromFS(nil, fs, "does-not-exist")
		require.Error(t, err)
	})

	t.Run("skip non-resource files in subdir", func(t *testing.T) {
		t.Parallel()
		fs := createTestFS(t)
		resources, err := fileconvert.ResourcesFromFS(func(_ string, _ ...any) {}, fs, "other")
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("read git repository end-to-end", func(t *testing.T) {
		t.Parallel()
		repoFs := memfs.New()
		repoStorer := memory.NewStorage()
		repo, err := git.Init(repoStorer, repoFs)
		require.NoError(t, err)

		w, err := repo.Worktree()
		require.NoError(t, err)

		f, err := repoFs.Create("profile.yaml")
		require.NoError(t, err)
		_, err = f.Write([]byte(profileYaml))
		require.NoError(t, err)
		f.Close()

		_, err = w.Add("profile.yaml")
		require.NoError(t, err)

		_, err = w.Commit("Initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})
		require.NoError(t, err)

		// Test ResourcesFromFS on the repo filesystem
		resources, err := fileconvert.ResourcesFromFS(nil, repoFs, "")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "test-profile", resources[0].GetName())
	})
}
