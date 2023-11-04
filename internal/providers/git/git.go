// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package git provides a client for interacting with Git providers
package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// Git is the struct that contains the GitHub REST API client
type Git struct {
	token string
}

// Ensure that the Git client implements the Git interface
var _ provifv1.Git = (*Git)(nil)

// NewGit creates a new GitHub client
func NewGit(token string) *Git {
	return &Git{
		token: token,
	}
}

// GetToken returns the token for the provider
func (g *Git) GetToken() string {
	return g.token
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

	if g.token != "" {
		opts.Auth = &http.BasicAuth{
			// the Username can be anything but it can't be empty
			Username: "minder-user",
			Password: g.token,
		}
	}

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid clone options: %w", err)
	}

	storer := memory.NewStorage()
	fs := memfs.New()

	// We clone to the memfs go-billy filesystem driver, which doesn't
	// allow for direct access to the underlying filesystem. This is
	// because we want to be able to run this in a sandboxed environment
	// where we don't have access to the underlying filesystem.
	r, err := git.CloneContext(ctx, storer, fs, opts)
	if err != nil {
		return nil, fmt.Errorf("could not clone repo: %w", err)
	}

	return r, nil
}
