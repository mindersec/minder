// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package scalibr

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/helper/iofs"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/stretchr/testify/require"
)

func TestScanFilesystem(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		makeFs      func() fs.FS
		mustErr     bool
		expect      *sbom.NodeList
		expectedLen int
	}{
		{
			name: "python-reqs-txt",
			makeFs: func() fs.FS {
				t.Helper()
				memFS := memfs.New()
				f, err := memFS.Create("requirements.txt")
				require.NoError(t, err)
				_, err = f.Write([]byte("Flask>=1\nrequestts>=1\n"))
				require.NoError(t, err)
				require.NoError(t, f.Close())
				return iofs.New(memFS)
			},
			expectedLen: 2,
			expect: &sbom.NodeList{
				Nodes: []*sbom.Node{
					{
						Id:      "0000000000",
						Type:    sbom.Node_PACKAGE,
						Name:    "Flask",
						Version: "1",
						Identifiers: map[int32]string{
							1: "pkg:pypi/flask@1",
						},
						Properties: []*sbom.Property{
							{
								Name: "sourceFile",
								Data: "requirements.txt",
							},
						},
					},
					{
						Id:      "1111111111",
						Type:    sbom.Node_PACKAGE,
						Name:    "requestts",
						Version: "1",
						Identifiers: map[int32]string{
							1: "pkg:pypi/requestts@1",
						},
						Properties: []*sbom.Property{
							{
								Name: "sourceFile",
								Data: "requirements.txt",
							},
						},
					},
				},
			},
		},
		{
			name: "bad-fs",
			makeFs: func() fs.FS {
				return nil
			},
			mustErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fs := tc.makeFs()
			nodelist, err := scanFilesystem(context.Background(), fs)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, nodelist.Nodes, tc.expectedLen)

			// Compare the nodes, make sure they are equal
			for i := range nodelist.Nodes {
				nodelist.Nodes[i].Id = strings.Repeat(fmt.Sprintf("%d", i), 10)
				require.Equal(t, tc.expect.Nodes[i].Checksum(), nodelist.Nodes[i].Checksum())
			}
		})
	}
}
