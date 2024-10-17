// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package git provides a client for interacting with Git providers
package git

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/mindersec/minder/internal/config/server"
	"github.com/mindersec/minder/internal/providers/git/memboxfs"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// Git is the struct that contains the GitHub REST API client
// It implements helper functions that a provider that
// uses the `git` trait can use.
type Git struct {
	credential provifv1.GitCredential
	maxFiles   int64
	maxBytes   int64
}

const maxCachedObjectSize = 100 * 1024 // 100KiB

// Options implements the "functional options" pattern for Git
type Options func(*Git)

// NewGit creates a new GitHub client
func NewGit(token provifv1.GitCredential, opts ...Options) *Git {
	ret := &Git{
		credential: token,
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}

// WithConfig configures the Git implementation with server-side configuration options.
func WithConfig(cfg server.GitConfig) Options {
	return func(g *Git) {
		g.maxFiles = cfg.MaxFiles
		g.maxBytes = cfg.MaxBytes
	}
}

// Clone clones a git repository
func (g *Git) Clone(ctx context.Context, url, branch string) (*git.Repository, error) {
	opts := &git.CloneOptions{
		URL:           url,
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.NoTags,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}

	g.credential.AddToCloneOptions(opts)

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid clone options: %w", err)
	}

	// TODO(#3582): Switch this to use a tmpfs backed clone
	memFS := memfs.New()
	if g.maxFiles != 0 && g.maxBytes != 0 {
		memFS = &memboxfs.LimitedFs{
			Fs:            memFS,
			MaxFiles:      g.maxFiles,
			TotalFileSize: g.maxBytes,
		}
	}
	// go-git seems to want separate filesystems for the storer and the checked out files
	storerFs := memfs.New()
	if g.maxFiles != 0 && g.maxBytes != 0 {
		storerFs = &memboxfs.LimitedFs{
			Fs:            storerFs,
			MaxFiles:      g.maxFiles,
			TotalFileSize: g.maxBytes,
		}
	}
	storerCache := cache.NewObjectLRU(maxCachedObjectSize)
	storer := filesystem.NewStorage(storerFs, storerCache)

	// We clone to the memfs go-billy filesystem driver, which doesn't
	// allow for direct access to the underlying filesystem. This is
	// because we want to be able to run this in a sandboxed environment
	// where we don't have access to the underlying filesystem.
	r, err := git.CloneContext(ctx, storer, memFS, opts)
	if err != nil {
		var refspecerr git.NoMatchingRefSpecError
		if errors.Is(err, git.ErrBranchNotFound) || refspecerr.Is(err) {
			return nil, provifv1.ErrProviderGitBranchNotFound
		} else if errors.Is(err, transport.ErrEmptyRemoteRepository) {
			return nil, provifv1.ErrRepositoryEmpty
		} else if errors.Is(err, memboxfs.ErrTooManyFiles) {
			return nil, fmt.Errorf("%w: %w", provifv1.ErrRepositoryTooLarge, err)
		} else if errors.Is(err, memboxfs.ErrTooBig) {
			return nil, fmt.Errorf("%w: %w", provifv1.ErrRepositoryTooLarge, err)
		}
		return nil, fmt.Errorf("could not clone repo: %w", err)
	}

	return r, nil
}
