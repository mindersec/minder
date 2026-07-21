// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package scalibr

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/helper/iofs"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
		expectedLog string
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
			name: "jumbo package lock (npm)",
			makeFs: func() fs.FS {
				t.Helper()
				memFS := memfs.New()
				f, err := memFS.Create("package-lock.json")
				require.NoError(t, err)
				fmt.Fprintf(f, `{"name":"test","version": "0.0.1", "lockfileVersion": 3, "requires": true, "packages": {`)
				// Ensure the package-lock.json is over 1MB
				for i := range 1000 {
					fmt.Fprintf(f, `"package-%d": {"resolved": "https://myregistry/@fake/package-%900d.tgz",`, i, i)
					fmt.Fprintf(f, ` "integrity": "sha512-00000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},`)
				}
				// We put the root package at the end so we don't have to mess with trailing commas
				fmt.Fprintf(f, `"": {"name": "test", "version": "0.0.1"}}}`)
				require.NoError(t, f.Close())
				return iofs.New(memFS)
			},
			expectedLen: 0,
			expectedLog: `"path":"package-lock.json","res":"FILE_REQUIRED_RESULT_SIZE_LIMIT_EXCEEDED"`,
		},
		{
			name: "go-binary does not panic on python",
			makeFs: func() fs.FS {
				t.Helper()
				memFS := memfs.New()
				f, err := memFS.Create("binary.py")
				require.NoError(t, err)
				fmt.Fprint(f, `print("hello world")\n`)
				require.NoError(t, f.Close())
				return iofs.New(memFS)
			},
			expectedLen: 0,
			expectedLog: `"plugin":"go/binary","path":"binary.py"`,
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
			logOutput := new(bytes.Buffer)
			ctx := zerolog.New(logOutput).WithContext(context.Background())
			nodelist, err := scanFilesystem(ctx, fs)
			if tc.mustErr {
				require.Error(t, err)
				return
			}

			assert.Contains(t, string(logOutput.String()), tc.expectedLog)
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
