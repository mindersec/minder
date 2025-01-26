// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package structured

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func writeFSFile(t *testing.T, fs billy.Filesystem, path string, data []byte) {
	t.Helper()
	require.NoError(t, fs.MkdirAll(filepath.Dir(path), os.FileMode(0o755)))
	f, err := fs.Create(path)
	require.NoError(t, err)
	_, err = f.Write(data)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestOpenFirstAlternative(t *testing.T) {
	t.Parallel()

	genFS := func(t *testing.T) billy.Filesystem {
		t.Helper()
		fs := memfs.New()
		writeFSFile(t, fs, "./test1.json", []byte("hello"))
		writeFSFile(t, fs, "/dir/test2.json", []byte("hello"))
		writeFSFile(t, fs, "dir2/test3.json", []byte("hello"))
		return fs
	}
	for _, tc := range []struct {
		name         string
		createFS     func(t *testing.T) billy.Filesystem
		mainPath     string
		alternatives []string
		expectedFile string
		mustErr      bool
	}{
		{
			name:         "mainpath",
			createFS:     genFS,
			mainPath:     "./test1.json",
			alternatives: []string{"dir/test2.json", "dir2/test.json"},
			expectedFile: "test1.json",
			mustErr:      false,
		},
		{
			name: "dir-must-be-ignored",
			createFS: func(t *testing.T) billy.Filesystem {
				t.Helper()
				fs := memfs.New()
				writeFSFile(t, fs, "./file.json", []byte("hello"))
				require.NoError(t, fs.MkdirAll("./dir", os.FileMode(0o755)))
				return fs
			},
			mainPath:     "./dir",
			alternatives: []string{"file.json"},
			expectedFile: "file.json",
			mustErr:      false,
		},
		{
			name:         "first-alternative",
			createFS:     genFS,
			mainPath:     "./non-existent",
			alternatives: []string{"dir/test2.json", "dir2/test.json"},
			expectedFile: "dir/test2.json",
			mustErr:      false,
		},
		{
			name:         "second-alternative",
			createFS:     genFS,
			mainPath:     "./non-existent2",
			alternatives: []string{"./non-existent", "dir2/test3.json"},
			expectedFile: "dir2/test3.json",
			mustErr:      false,
		},
		{
			name:         "no-main",
			createFS:     genFS,
			mainPath:     "",
			alternatives: []string{"dir2/test3.json"},
			expectedFile: "dir2/test3.json",
			mustErr:      false,
		},
		{
			name:         "no-valid-files",
			createFS:     genFS,
			mainPath:     "non-existing",
			alternatives: []string{"also-non-existing.txt"},
			expectedFile: "",
			mustErr:      true,
		},
		{
			name:         "no-inputs",
			createFS:     genFS,
			mainPath:     "",
			alternatives: []string{},
			expectedFile: "",
			mustErr:      true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fs := tc.createFS(t)
			f, err := openFirstAlternative(fs, tc.mainPath, tc.alternatives)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedFile, f.Name())
		})
	}
}

func TestParseFileAlternatives(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		main    string
		mustErr bool
	}{
		{"fn-success", "test1.json", false},
		{"fn-fails", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fs := memfs.New()
			writeFSFile(t, fs, "./test1.json", []byte("{ \"a\": \"b\"}"))
			res, err := parseFileAlternatives(fs, tc.main, []string{})
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	h, err := newHandlerFromDef(&minderv1.StructDataSource_Def{
		Path: &minderv1.StructDataSource_Def_Path{
			FileName: "test.txt",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, h)

	_, err = newHandlerFromDef(nil)
	require.Error(t, err)
}

func TestCall(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		ingest  func(t *testing.T) *interfaces.Result
		def     *minderv1.StructDataSource_Def
		mustErr bool
	}{
		{
			"success",
			func(t *testing.T) *interfaces.Result {
				t.Helper()
				fs := memfs.New()
				writeFSFile(t, fs, "./test1.json", []byte("{ \"a\": \"b\"}"))

				return &interfaces.Result{Fs: fs}
			},
			&minderv1.StructDataSource_Def{
				Path: &minderv1.StructDataSource_Def_Path{
					FileName: "test1.json",
				},
			},
			false,
		},
		{
			"no-datasource-context",
			func(t *testing.T) *interfaces.Result {
				t.Helper()
				return nil
			},
			&minderv1.StructDataSource_Def{},
			true,
		},
		{"ctx-no-fs",
			func(t *testing.T) *interfaces.Result {
				t.Helper()
				return &interfaces.Result{}
			},
			&minderv1.StructDataSource_Def{},
			true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ingest := tc.ingest(t)
			handler, err := newHandlerFromDef(tc.def)
			require.NoError(t, err)
			_, err = handler.Call(context.Background(), ingest, []string{})
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
