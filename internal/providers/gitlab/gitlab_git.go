// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"

	"github.com/go-git/go-git/v5"

	gitclient "github.com/mindersec/minder/internal/providers/git"
)

// Implements the Git interface
func (c *gitlabClient) Clone(ctx context.Context, cloneUrl string, branch string) (*git.Repository, error) {
	g := gitclient.NewGit(c.GetCredential(), gitclient.WithConfig(c.gitConfig))
	return g.Clone(ctx, cloneUrl, branch)
}
