// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package mindpak

import (
	"io/fs"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestReadSource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		path       string
		mustErr    bool
		filesMatch bool
		expect     *Bundle
	}{
		{
			"normal",
			"testdata/t1",
			false,
			true,
			&Bundle{
				Manifest: &Manifest{},
				Metadata: &Metadata{},
				Files: &Files{
					Profiles: []*File{
						{
							Name:   "branch-protection.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "21e74a8d380c2940b0b26798f7ba7a5236b5444b02ff0bf45ce28f0016a24f65"},
						},
					},
					RuleTypes: []*File{
						{
							Name:   "secret_scanning.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "3857bca2ccabdac3d136eb3df4549ddd87a00ddef9fdcf88d8f824e5e796d34c"},
						},
					},
				},
				Source: nil,
			},
		},
		{
			"wrong-hash",
			"testdata/t1",
			false,
			false,
			&Bundle{
				Manifest: &Manifest{},
				Metadata: &Metadata{},
				Files: &Files{
					Profiles: []*File{
						{
							Name:   "branch-protection.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "AAf3682a1cb5ab92c0cc71dd913338bf40a89ec324024f8d3f500be0e2aa4a9ae1"},
						},
					},
					RuleTypes: []*File{
						{
							Name:   "secret_scanning.yaml",
							Hashes: map[HashAlgorithm]string{SHA256: "AA572089a9a490d1b7d07f2a1f6845ae1f18af27a6a13a605de7cef8a910427084"},
						},
					},
				},
				Source: nil,
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := Bundle{}
			m.Source = os.DirFS(tc.path).(fs.StatFS)
			err := m.ReadSource()
			if tc.mustErr {
				require.Error(t, err)
				return
			}

			diff := cmp.Diff(&tc.expect.Files, &m.Files, protocmp.Transform())
			if tc.filesMatch {
				require.Empty(t, diff, "file hashes don't match:\n%v", diff)
			} else {
				require.NotEmpty(t, diff, "file hashes should not match")
			}

		})

	}
}
