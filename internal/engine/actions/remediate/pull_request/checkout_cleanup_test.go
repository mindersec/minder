// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestCheckoutToOriginallyFetchedBranch_CleansWorktree(t *testing.T) {
	t.Parallel()

	mfs := memfs.New()
	storer := memory.NewStorage()
	repo, err := git.InitWithOptions(storer, mfs, git.InitOptions{
		DefaultBranch: "refs/heads/main",
	})
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create an initial commit on main so we have a valid HEAD.
	f, err := mfs.Create("README.md")
	require.NoError(t, err)
	_, err = f.Write([]byte("initial"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = wt.Add("README.md")
	require.NoError(t, err)
	_, err = wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "t@t.com", When: time.Now()},
	})
	require.NoError(t, err)

	mainRef := plumbing.NewBranchReferenceName("main")

	// Checkout a remediation branch and simulate a partial failure:
	// modify a tracked file and create an untracked file without committing.
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("minder_remediation"),
		Create: true,
	})
	require.NoError(t, err)

	// Dirty a tracked file (simulates remediation writing a compliant version).
	f, err = mfs.Create("README.md")
	require.NoError(t, err)
	_, err = f.Write([]byte("dirty content from failed remediation"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// Create an untracked file (simulates remediation creating a new config).
	f, err = mfs.Create("leftover.txt")
	require.NoError(t, err)
	_, err = f.Write([]byte("should be removed"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// Verify the worktree is actually dirty before we clean it.
	status, err := wt.Status()
	require.NoError(t, err)
	require.False(t, status.IsClean(), "worktree should be dirty before cleanup")

	// Run checkoutToOriginallyFetchedBranch — this is the code under test.
	logger := zerolog.Nop()
	checkoutToOriginallyFetchedBranch(&logger, wt, mainRef)

	// Assert: worktree should now be clean.
	status, err = wt.Status()
	require.NoError(t, err)
	require.True(t, status.IsClean(), "worktree should be clean after checkout with Force + Clean")

	// Assert: untracked file should have been removed by wt.Clean.
	_, err = mfs.Stat("leftover.txt")
	require.Error(t, err, "untracked file should have been removed by Clean")
}
