// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/go-git/go-billy/v5/helper/iofs"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ParseGitURL parses a repository URL and optional subdirectory selector.
// The subdirectory is specified after a '?' delimiter (e.g. "https://github.com/org/repo?sub/dir").
func ParseGitURL(rawURL string) (repoURL string, subDir string, err error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", "", errors.New("git URL cannot be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid git URL: %w", err)
	}

	if u.Scheme == "" || (u.Scheme != "file" && u.Host == "") {
		return "", "", errors.New("git URL must include scheme and host")
	}

	subDir = u.RawQuery
	subDir = strings.TrimPrefix(subDir, "/")
	subDir = path.Clean(subDir)
	if subDir == "." {
		subDir = ""
	}

	// Reconstruct the base repository URL without query or fragment
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	repoURL = u.String()

	return repoURL, subDir, nil
}

// ResourcesFromGit clones a remote git repository into memory and collects all Minder resources
// found at the specified subdirectory (or repo root if none).
func ResourcesFromGit(ctx context.Context, printer Printer, rawURL string) ([]minderv1.ResourceMeta, error) {
	repoURL, subDir, err := ParseGitURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing git URL: %w", err)
	}

	fs := memfs.New()
	storer := memory.NewStorage()

	cloneOpts := &git.CloneOptions{
		URL:          repoURL,
		Depth:        1,
		SingleBranch: true,
		Tags:         git.NoTags,
	}

	if _, err := git.CloneContext(ctx, storer, fs, cloneOpts); err != nil {
		return nil, fmt.Errorf("error cloning git repository %s: %w", repoURL, err)
	}

	return ResourcesFromPaths(ctx, iofs.New(fs), printer, subDir)
}
