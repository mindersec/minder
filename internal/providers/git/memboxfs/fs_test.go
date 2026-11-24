// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package memboxfs

import (
	"io"
	"os"
	"testing"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLimitedFs(maxFiles, maxBytes int64) *LimitedFs {
	return &LimitedFs{
		Fs:            memfs.New(),
		MaxFiles:      maxFiles,
		TotalFileSize: maxBytes,
	}
}

// TestLimitedFs_FileCountLimit checks that every operation which adds an entry
// to the filesystem is subject to the MaxFiles limit, and that the returned
// error can be matched with errors.Is (the git provider relies on this to map
// the failure to ErrRepositoryTooLarge). Exactly MaxFiles entries are allowed.
func TestLimitedFs_FileCountLimit(t *testing.T) {
	t.Parallel()

	const maxFiles = 3

	scenarios := []struct {
		name string
		add  func(lfs *LimitedFs, i int) error
	}{
		{
			name: "Create",
			add: func(lfs *LimitedFs, i int) error {
				f, err := lfs.Create(fileName(i))
				if err != nil {
					return err
				}
				return f.Close()
			},
		},
		{
			name: "OpenFile with O_CREATE",
			add: func(lfs *LimitedFs, i int) error {
				f, err := lfs.OpenFile(fileName(i), os.O_CREATE|os.O_WRONLY, 0o644)
				if err != nil {
					return err
				}
				return f.Close()
			},
		},
		{
			name: "TempFile",
			add: func(lfs *LimitedFs, _ int) error {
				f, err := lfs.TempFile("tmp", "prefix")
				if err != nil {
					return err
				}
				return f.Close()
			},
		},
		{
			name: "MkdirAll",
			add: func(lfs *LimitedFs, i int) error {
				return lfs.MkdirAll(fileName(i), 0o755)
			},
		},
		{
			name: "Symlink",
			add: func(lfs *LimitedFs, i int) error {
				return lfs.Symlink("target", fileName(i))
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()

			lfs := newLimitedFs(maxFiles, 1024)

			for i := 0; i < maxFiles; i++ {
				require.NoError(t, sc.add(lfs, i), "entry %d should be within the limit", i)
			}

			err := sc.add(lfs, maxFiles)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrTooManyFiles)
		})
	}
}

func TestLimitedFs_OpenFileWithoutCreateDoesNotCount(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(3, 1024)

	f, err := lfs.Create("existing")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// Re-opening an existing file without O_CREATE does not add an entry, so it
	// must never hit the file-count limit no matter how often it happens.
	for i := 0; i < 10; i++ {
		f, err := lfs.OpenFile("existing", os.O_RDWR, 0o644)
		require.NoError(t, err, "re-open %d", i)
		require.NoError(t, f.Close())

		f, err = lfs.Open("existing")
		require.NoError(t, err, "open %d", i)
		require.NoError(t, f.Close())
	}
}

func TestLimitedFs_WriteSizeLimit(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name      string
		maxBytes  int64
		writes    []string
		wantErrAt int // index of the write expected to fail, or -1 for none
	}{
		{
			name:      "single write within limit",
			maxBytes:  10,
			writes:    []string{"hello"},
			wantErrAt: -1,
		},
		{
			name:      "single write exactly at limit",
			maxBytes:  5,
			writes:    []string{"hello"},
			wantErrAt: -1,
		},
		{
			name:      "single write over limit",
			maxBytes:  4,
			writes:    []string{"hello"},
			wantErrAt: 0,
		},
		{
			name:      "appending writes exceed limit",
			maxBytes:  8,
			writes:    []string{"hello", "world"},
			wantErrAt: 1,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()

			lfs := newLimitedFs(100, sc.maxBytes)
			f, err := lfs.Create("file")
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })

			for i, w := range sc.writes {
				n, err := f.Write([]byte(w))
				if i == sc.wantErrAt {
					require.ErrorIs(t, err, ErrTooBig)
					assert.Zero(t, n)
					return
				}
				require.NoError(t, err, "write %d", i)
				assert.Equal(t, len(w), n)
			}
		})
	}
}

func TestLimitedFs_SizeLimitIsSharedAcrossFiles(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(100, 8)

	a, err := lfs.Create("a")
	require.NoError(t, err)
	defer a.Close()
	b, err := lfs.Create("b")
	require.NoError(t, err)
	defer b.Close()

	_, err = a.Write([]byte("hello"))
	require.NoError(t, err)

	// 5 bytes already used by "a"; another 5 in "b" exceeds the 8 byte budget.
	_, err = b.Write([]byte("world"))
	require.ErrorIs(t, err, ErrTooBig)
}

func TestLimitedFs_RejectedWriteDoesNotConsumeBudget(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(100, 8)
	f, err := lfs.Create("file")
	require.NoError(t, err)
	defer f.Close()

	_, err = f.Write([]byte("this is far too long"))
	require.ErrorIs(t, err, ErrTooBig)

	// The failed write must not have been charged against the budget.
	n, err := f.Write([]byte("12345678"))
	require.NoError(t, err)
	assert.Equal(t, 8, n)
}

func TestLimitedFs_OverwriteDoesNotConsumeBudget(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(100, 5)
	f, err := lfs.Create("file")
	require.NoError(t, err)
	defer f.Close()

	_, err = f.Write([]byte("hello"))
	require.NoError(t, err)

	// Rewriting the same bytes in place does not grow the file, so it must be
	// allowed even though the budget is fully used.
	for i := 0; i < 3; i++ {
		_, err = f.Seek(0, io.SeekStart)
		require.NoError(t, err)
		_, err = f.Write([]byte("HELLO"))
		require.NoError(t, err, "overwrite %d", i)
	}

	// Growing past the end still fails.
	_, err = f.Write([]byte("!"))
	require.ErrorIs(t, err, ErrTooBig)

	assertContent(t, lfs, "file", "HELLO")
}

func TestLimitedFs_TruncateSizeLimit(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(100, 10)
	f, err := lfs.Create("file")
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, f.Truncate(10), "growing up to the limit is allowed")
	require.ErrorIs(t, f.Truncate(11), ErrTooBig, "growing past the limit is rejected")

	info, err := lfs.Stat("file")
	require.NoError(t, err)
	assert.Equal(t, int64(10), info.Size(), "rejected truncate must not change the file")
}

func TestLimitedFs_ChrootNotImplemented(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(10, 10)
	got, err := lfs.Chroot("dir")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrNotImplemented)
}

func TestLimitedFs_PassThroughOperations(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(100, 1024)

	assert.Equal(t, lfs.Fs.Root(), lfs.Root())
	assert.Equal(t, lfs.Fs.Join("a", "b", "c"), lfs.Join("a", "b", "c"))

	require.NoError(t, lfs.MkdirAll("dir", 0o755))
	writeFile(t, lfs, "dir/file", "content")

	entries, err := lfs.ReadDir("dir")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "file", entries[0].Name())

	info, err := lfs.Lstat("dir/file")
	require.NoError(t, err)
	assert.Equal(t, int64(len("content")), info.Size())

	// Use a target without path separators: memfs stores link targets using
	// the OS separator, which would make the assertion platform-dependent.
	require.NoError(t, lfs.Symlink("target", "link"))
	target, err := lfs.Readlink("link")
	require.NoError(t, err)
	assert.Equal(t, "target", target)

	require.NoError(t, lfs.Rename("dir/file", "dir/renamed"))
	assertContent(t, lfs, "dir/renamed", "content")

	require.NoError(t, lfs.Remove("dir/renamed"))
	_, err = lfs.Stat("dir/renamed")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestFileWrapper_ReadOperations(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(100, 1024)
	writeFile(t, lfs, "file", "hello world")

	f, err := lfs.OpenFile("file", os.O_RDWR, 0o644)
	require.NoError(t, err)
	defer f.Close()

	assert.Equal(t, "file", f.Name())

	buf := make([]byte, 5)
	n, err := f.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(buf[:n]))

	n, err = f.ReadAt(buf, 6)
	require.NoError(t, err)
	assert.Equal(t, "world", string(buf[:n]))

	pos, err := f.Seek(0, io.SeekCurrent)
	require.NoError(t, err)
	assert.Equal(t, int64(5), pos, "ReadAt must not move the file offset")

	require.NoError(t, f.Lock())
	require.NoError(t, f.Unlock())
}

func fileName(i int) string {
	return "entry-" + string(rune('a'+i))
}

func writeFile(t *testing.T, lfs billy.Filesystem, name, content string) {
	t.Helper()
	f, err := lfs.Create(name)
	require.NoError(t, err)
	_, err = f.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func assertContent(t *testing.T, lfs billy.Filesystem, name, want string) {
	t.Helper()
	f, err := lfs.Open(name)
	require.NoError(t, err)
	defer f.Close()
	got, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

// TestLimitedFs_FailedCreateDoesNotCount checks that when the underlying
// filesystem fails to create an entry, no file is returned and the entry is
// not counted against MaxFiles.
func TestLimitedFs_FailedCreateDoesNotCount(t *testing.T) {
	t.Parallel()

	lfs := newLimitedFs(1, 1024)
	require.NoError(t, lfs.Fs.MkdirAll("dir", 0o755))

	// "dir" is a directory, so creating a file with that name fails.
	f, err := lfs.Create("dir")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrTooManyFiles)
	assert.Nil(t, f)

	f, err = lfs.OpenFile("dir", os.O_CREATE|os.O_WRONLY, 0o644)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrTooManyFiles)
	assert.Nil(t, f)

	// The failed creates didn't use up the single allowed entry.
	f, err = lfs.Create("file")
	require.NoError(t, err)
	require.NoError(t, f.Close())
}
