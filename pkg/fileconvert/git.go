// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"gopkg.in/yaml.v3"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ParseGitURL parses a repository URL and optional subdirectory selector.
// The subdirectory is specified after a '?' delimiter (e.g. "https://github.com/org/repo?sub/dir").
func ParseGitURL(rawURL string) (repoURL string, subDir string, err error) {
	if rawURL == "" {
		return "", "", errors.New("git URL cannot be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid git URL: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
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

	return ResourcesFromFS(printer, fs, subDir)
}

// ResourcesFromFS reads all Minder resources from a billy.Filesystem starting at targetPath.
func ResourcesFromFS(printer Printer, fs billy.Filesystem, targetPath string) ([]minderv1.ResourceMeta, error) {
	cleanPath := path.Clean(targetPath)
	if cleanPath == "." {
		cleanPath = ""
	}

	fi, err := fs.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("error accessing path %q in repository: %w", targetPath, err)
	}

	var objects []minderv1.ResourceMeta

	if !fi.IsDir() {
		// Single file
		res, err := readFromFSPath(printer, fs, cleanPath, false)
		if err != nil && !errors.Is(err, minderv1.ErrNotAResource) {
			return nil, err
		}
		return res, nil
	}

	// Directory traversal
	err = billyutil.Walk(fs, cleanPath, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error walking path %s: %w", filePath, walkErr)
		}
		if info.IsDir() {
			return nil
		}

		res, err := readFromFSPath(printer, fs, filePath, true)
		if err != nil {
			if errors.Is(err, minderv1.ErrNotAResource) {
				return nil
			}
			return err
		}
		objects = append(objects, res...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error traversing repository files: %w", err)
	}

	return objects, nil
}

func readFromFSPath(printer Printer, fs billy.Filesystem, filePath string, isExpanded bool) ([]minderv1.ResourceMeta, error) {
	ext := path.Ext(filePath)
	var input Decoder

	file, err := fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer file.Close()

	switch ext {
	case ".json":
		input = yaml.NewDecoder(file) // yaml.NewDecoder can decode JSON safely
	case ".yaml", ".yml":
		input = yaml.NewDecoder(file)
	case ".rego":
		input = &regoDecoder{filename: filePath, file: file}
	default:
		// Not a recognized file extension
		return nil, nil
	}

	var objects []minderv1.ResourceMeta
	for i := 0; ; i++ {
		resource, err := ReadResource(input)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if isExpanded && i == 0 {
				if printer != nil {
					printer("Skipping file %s due to error: %s\n", filePath, err)
				}
				break
			}
			return nil, fmt.Errorf("error reading resource from %s: %w", filePath, err)
		}
		objects = append(objects, resource)
	}

	return objects, nil
}
