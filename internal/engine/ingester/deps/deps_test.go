// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/stretchr/testify/require"

	mock_github "github.com/mindersec/minder/internal/providers/github/mock"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestScanFS(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		makeFs      func() billy.Filesystem
		mustErr     bool
		expect      *sbom.NodeList
		expectedLen int
	}{
		{
			name: "python-reqs-txt",
			makeFs: func() billy.Filesystem {
				t.Helper()
				memFS := memfs.New()
				f, err := memFS.Create("requirements.txt")
				require.NoError(t, err)
				_, err = f.Write([]byte("Flask>=1\nrequestts>=1\n"))
				require.NoError(t, err)
				require.NoError(t, f.Close())
				return memFS
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
			makeFs: func() billy.Filesystem {
				return nil
			},
			mustErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fs := tc.makeFs()
			nodelist, err := scanFs(context.Background(), fs)
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

func TestGetBranch(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		repo         *v1.Repository
		branch       string
		configBranch string
		expect       string
	}{
		{name: "default", expect: "main"},
		{name: "branch", branch: "test1", expect: "test1"},
		{name: "repo-default", repo: &v1.Repository{DefaultBranch: "defaultBranch"}, expect: "defaultBranch"},
		{name: "repo-default", configBranch: "ingestBranch", expect: "ingestBranch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gi, err := NewDepsIngester(&v1.DepsType{
				EntityType: &v1.DepsType_Repo{
					Repo: &v1.DepsType_RepoConfigs{
						Branch: tc.configBranch,
					},
				},
			}, &mock_github.MockGit{})
			require.NoError(t, err)

			branch := gi.getBranch(tc.repo, tc.branch)
			require.Equal(t, tc.expect, branch)
		})
	}
}
