// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package build implements tools and function to build mindpaks. The main
// builder is build.Packer that writes the bundles to archives.

package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/mindpak"
)

func TestPackerInitBundle(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		opts    *InitOptions
		prepare func(*testing.T, *InitOptions)
		mustErr bool
	}{
		{
			name: "nofiles",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name: "my-bundle",
				},
			},
			prepare: func(t *testing.T, opts *InitOptions) {
				t.Helper()
				d := t.TempDir()
				opts.Path = d
			},
		},
		{
			name: "nofiles-withnamespace",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "my-bundle",
					Namespace: "ns",
				},
			},
			prepare: func(t *testing.T, opts *InitOptions) {
				t.Helper()
				d := t.TempDir()
				opts.Path = d
			},
		},
		{
			name: "files",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "my-bundle",
					Namespace: "ns",
				},
			},
			prepare: func(t *testing.T, opts *InitOptions) {
				t.Helper()
				d := t.TempDir()
				opts.Path = d

				require.NoError(t, os.Mkdir(filepath.Join(opts.Path, mindpak.PathProfiles), os.FileMode(0o700)))
				require.NoError(t, os.Mkdir(filepath.Join(opts.Path, mindpak.PathRuleTypes), os.FileMode(0o700)))

				require.NoError(t, os.WriteFile(filepath.Join(opts.Path, mindpak.PathProfiles, "test1"), []byte("test"), os.FileMode(0o644)))
				require.NoError(t, os.WriteFile(filepath.Join(opts.Path, mindpak.PathRuleTypes, "test2"), []byte("test2"), os.FileMode(0o644)))
			},
		},
		{
			name: "unexpected-files",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "my-bundle",
					Namespace: "ns",
				},
			},
			prepare: func(t *testing.T, opts *InitOptions) {
				t.Helper()
				d := t.TempDir()
				opts.Path = d

				require.NoError(t, os.Mkdir(filepath.Join(opts.Path, mindpak.PathProfiles), os.FileMode(0o700)))
				require.NoError(t, os.Mkdir(filepath.Join(opts.Path, mindpak.PathRuleTypes), os.FileMode(0o700)))

				require.NoError(t, os.WriteFile(filepath.Join(opts.Path, mindpak.PathProfiles, "test1"), []byte("test"), os.FileMode(0o644)))
				require.NoError(t, os.WriteFile(filepath.Join(opts.Path, mindpak.PathRuleTypes, "test2"), []byte("test2"), os.FileMode(0o644)))
				require.NoError(t, os.WriteFile(filepath.Join(opts.Path, "hola"), []byte("test3"), os.FileMode(0o644)))
			},
			mustErr: true,
		},
		{
			name:    "noopts",
			prepare: func(_ *testing.T, _ *InitOptions) {},
			mustErr: true,
		},
		{
			name:    "noname",
			opts:    &InitOptions{},
			prepare: func(_ *testing.T, _ *InitOptions) {},
			mustErr: true,
		},
		{
			name: "invalid-dir",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name: "my-bundle",
				},
				Path: "my-dir",
			},
			prepare: func(_ *testing.T, _ *InitOptions) {},
			mustErr: true,
		},
	} {
		t.Run(t.Name(), func(t *testing.T) {
			t.Parallel()

			tc.prepare(t, tc.opts)
			p := NewPacker()

			// Run the bundle initialization
			_, err := p.InitBundle(tc.opts)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.FileExists(t, filepath.Join(tc.opts.Path, mindpak.ManifestFileName))
		})
	}
}

func TestValidateInitOpts(t *testing.T) {
	tmp := t.TempDir()
	now := time.Now()
	t.Parallel()
	for _, tc := range []struct {
		name      string
		opts      *InitOptions
		shouldErr bool
	}{
		{
			name: "noerror",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "my-bundle",
					Namespace: "ns",
					Version:   "1.0.0",
					Date:      &now,
				},
				Path: tmp,
			},
			shouldErr: false,
		},
		{
			name: "noname",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "",
					Namespace: "ns",
					Version:   "1.0.0",
					Date:      &now,
				},
				Path: tmp,
			},
			shouldErr: true,
		},
		{
			name: "invalid name",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "it an invalid name!",
					Namespace: "ns",
					Version:   "1.0.0",
					Date:      &now,
				},
				Path: tmp,
			},
			shouldErr: true,
		},
		{
			name: "invalid namespace",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "name",
					Namespace: "it an invalid namespace!",
					Version:   "1.0.0",
					Date:      &now,
				},
				Path: tmp,
			},
			shouldErr: true,
		},
		{
			name: "dir-notexists",
			opts: &InitOptions{
				Metadata: &mindpak.Metadata{
					Name:      "name",
					Namespace: "ns",
					Version:   "1.0.0",
					Date:      &now,
				},
				Path: "jklsdkjlsdljk sdkjl sd jkldsjkl",
			},
			shouldErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.opts.Validate()
			if tc.shouldErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
