// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package memboxfs

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/require"
)

func newTestLimitedFs(maxFiles int64) *LimitedFs {
	return &LimitedFs{
		Fs:            memfs.New(),
		MaxFiles:      maxFiles,
		TotalFileSize: 1024,
	}
}

func TestMkdirAllCountsOnlyMissingSegments(t *testing.T) {
	t.Parallel()

	fsys := newTestLimitedFs(100)

	// Three new path segments: /a, /a/b and /a/b/c.
	require.NoError(t, fsys.MkdirAll("/a/b/c", 0o755))
	require.Equal(t, int64(3), fsys.currentFiles)

	// Re-creating the same tree adds nothing to the budget.
	require.NoError(t, fsys.MkdirAll("/a/b/c", 0o755))
	require.Equal(t, int64(3), fsys.currentFiles)

	// A sibling dir under existing parents is a single new segment.
	require.NoError(t, fsys.MkdirAll("/a/b/d", 0o755))
	require.Equal(t, int64(4), fsys.currentFiles)
}

func TestMkdirAllRejectionLeavesBudgetAndFsUntouched(t *testing.T) {
	t.Parallel()

	fsys := newTestLimitedFs(3)

	// /a, /a/b, /a/b/c would need 3 slots and 0+3 >= 3, so this must fail
	// and consume nothing.
	err := fsys.MkdirAll("/a/b/c", 0o755)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrTooManyFiles))
	require.Equal(t, int64(0), fsys.currentFiles)

	_, err = fsys.Stat("/a")
	require.ErrorIs(t, err, fs.ErrNotExist)

	// The counter was not incremented by the failed call: a single-segment
	// mkdir (0+1 < 3) still succeeds afterwards.
	require.NoError(t, fsys.MkdirAll("/x", 0o755))
	require.Equal(t, int64(1), fsys.currentFiles)
}

func TestMkdirAllOverExistingFileCountsOnlyMissingSegment(t *testing.T) {
	t.Parallel()

	fsys := newTestLimitedFs(100)

	file, err := fsys.OpenFile("/file", os.O_CREATE|os.O_RDWR, 0o644)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, int64(1), fsys.currentFiles)

	// /file/dir has exactly one missing segment: the walk stops at the
	// existing /file ancestor regardless of its type. (memfs is a flat
	// map without POSIX ENOTDIR semantics, so the delegated MkdirAll
	// succeeds; what LimitedFs must guarantee is that only the genuinely
	// missing segment consumes the budget.)
	require.NoError(t, fsys.MkdirAll("/file/dir", 0o755))
	require.Equal(t, int64(2), fsys.currentFiles)
}

func TestCreateStillCountsTowardLimit(t *testing.T) {
	t.Parallel()

	fsys := newTestLimitedFs(3)

	for _, name := range []string{"/a", "/b"} {
		file, err := fsys.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o644)
		require.NoError(t, err)
		require.NoError(t, file.Close())
	}
	require.Equal(t, int64(2), fsys.currentFiles)

	// Sibling Create semantics are unchanged: the counter is incremented
	// first and calls that reach MaxFiles are rejected.
	_, err := fsys.OpenFile("/c", os.O_CREATE|os.O_RDWR, 0o644)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrTooManyFiles))
}
