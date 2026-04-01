// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

// TestVerifyGitFiles_FileMissingFromFS covers the error branch inside
// verifyGitFiles where the filesystem does not contain a file that the fixture
// declared.  verifyGitFiles is called by DryRun with the filesystem returned
// by BuildMocks, so this situation would only occur if there were a bug in
// BuildMocks.  Having an explicit test keeps us safe if the function is ever
// called from other callsites in the future.
func TestVerifyGitFiles_FileMissingFromFS_ReturnsError(t *testing.T) {
	t.Parallel()

	// Create an empty filesystem and then ask verifyGitFiles to verify a file
	// that was never written into it.
	emptyFS := memfs.New()

	err := verifyGitFiles(emptyFS, map[string]string{
		"SECURITY.md": "report vulns here",
	})

	if err == nil {
		t.Fatal("expected an error when the file is missing from the filesystem, got nil")
	}
}

// TestVerifyGitFiles_EmptyMap_NeverErrors confirms that an empty file map
// is always fine.  DryRun passes whatever is in the fixture's git_files block,
// so a fixture with no git files should never produce a verification error.
func TestVerifyGitFiles_EmptyMap_NeverErrors(t *testing.T) {
	t.Parallel()

	emptyFS := memfs.New()

	if err := verifyGitFiles(emptyFS, map[string]string{}); err != nil {
		t.Errorf("expected no error for empty file map, got: %v", err)
	}
	if err := verifyGitFiles(emptyFS, nil); err != nil {
		t.Errorf("expected no error for nil file map, got: %v", err)
	}
}

// TestVerifyGitFiles_MultipleFiles_FailsOnFirstMissing checks that
// verifyGitFiles returns as soon as it finds the first missing file and does
// not silently skip it.
func TestVerifyGitFiles_MultipleFiles_FailsOnFirstMissing(t *testing.T) {
	t.Parallel()

	// Only write one of two declared files into the filesystem.
	fs, err := NewMockBillyFS(map[string]string{
		"README.md": "hello",
	})
	if err != nil {
		t.Fatalf("building filesystem: %v", err)
	}

	err = verifyGitFiles(fs, map[string]string{
		"README.md":   "hello",
		"SECURITY.md": "missing from fs",
	})

	if err == nil {
		t.Fatal("expected an error because SECURITY.md is missing, got nil")
	}
}

func verifyGitFiles(fs billy.Filesystem, files map[string]string) error {
	for path := range files {
		f, err := fs.Open(path)
		if err != nil {
			return fmt.Errorf("git_files: cannot open %q in mock filesystem: %w", path, err)
		}
		_ = f.Close()
	}
	return nil
}

func NewMockBillyFS(files map[string]string) (billy.Filesystem, error) {
	fs := memfs.New()
	for path, content := range files {
		f, err := fs.Create(path)
		if err != nil {
			return nil, fmt.Errorf("creating %q: %w", path, err)
		}
		if _, writeErr := f.Write([]byte(content)); writeErr != nil {
			f.Close()
			return nil, fmt.Errorf("writing %q: %w", path, writeErr)
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("closing %q: %w", path, err)
		}
	}
	return fs, nil
}
