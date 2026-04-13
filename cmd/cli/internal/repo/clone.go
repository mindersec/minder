// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package repo provides helpers to interact with git repositories.
package repo

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

// CloneInMemory clones a git repository into memory.
//
// This is a preparatory utility for future enhancements where
// quickstart will dynamically load rule and profile catalogs
// from a repository without touching disk.
func CloneInMemory(url string) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: url,
	})
}
