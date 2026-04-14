// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package cli contains reusable CLI helper logic for catalog operations
package cli

import (
	"fmt"

	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

// CloneRepoFilesystem clones the given Git repository URL into an in-memory
// storage and returns its filesystem. Callers can use the returned
// filesystem to open files and traverse directories without touching disk.
func CloneRepoFilesystem(repoURL string) (billy.Filesystem, error) {
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:   repoURL,
		Depth: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository %s: %w", repoURL, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree for %s: %w", repoURL, err)
	}

	return worktree.Filesystem, nil
}
