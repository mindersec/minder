// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/prototext"

	mock_github "github.com/mindersec/minder/internal/providers/github/mock"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

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

func TestSBOMNodeDiff(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		op     string
		base   []*sbom.Node
		target []*sbom.Node
		expect []*sbom.Node
	}{{
		name:   "nil",
		op:     "new",
		expect: []*sbom.Node{},
	}, {
		name:   "nil + all",
		op:     "all",
		expect: []*sbom.Node{},
	}, {
		name: "equal, new",
		op:   "new",
		base: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3"},
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
		},
		target: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3"},
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
		},
		expect: []*sbom.Node{},
	}, {
		name: "equal, all",
		op:   "all",
		base: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3"},
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
		},
		target: []*sbom.Node{
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
			{Id: "1", Name: "pandas", Version: "2.2.3"},
		},
		expect: []*sbom.Node{
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
			{Id: "1", Name: "pandas", Version: "2.2.3"},
		},
	}, {
		name: "different versions only, new",
		op:   "new",
		base: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3"},
			{Id: "2", Name: "fastapi", Version: "0.115.4"},
		},
		target: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3"},
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
			{Id: "3", Name: "pandas", Version: "2.2.9"},
		},
		expect: []*sbom.Node{},
	}, {
		name: "different versions, new_and_updated",
		op:   "new_and_updated",
		base: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.1.1"},
			{Id: "2", Name: "fastapi", Version: "0.102.4"},
		},
		target: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3"},
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
		},
		expect: []*sbom.Node{
			// Output is alphabetical by name, then version
			{Id: "2", Name: "fastapi", Version: "0.115.6"},
			{Id: "1", Name: "pandas", Version: "2.2.3"},
		},
	}, {
		name: "updated hashes, new_and_updated",
		op:   "new_and_updated",
		base: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3",
				Hashes: map[int32]string{int32(sbom.HashAlgorithm_SHA1): "abc"}},
			{Id: "2", Name: "fastapi", Version: "0.115.6",
				Hashes: map[int32]string{int32(sbom.HashAlgorithm_SHA1): "def"}},
		},
		target: []*sbom.Node{
			{Id: "1", Name: "pandas", Version: "2.2.3",
				Hashes: map[int32]string{int32(sbom.HashAlgorithm_SHA1): "abc"}},
			{Id: "2", Name: "fastapi", Version: "0.115.6",
				Hashes: map[int32]string{int32(sbom.HashAlgorithm_SHA1): "aou"}},
		},
		expect: []*sbom.Node{
			{Id: "2", Name: "fastapi", Version: "0.115.6",
				Hashes: map[int32]string{int32(sbom.HashAlgorithm_SHA1): "aou"}},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			selected := filterNodes(tc.base, tc.target, ingestTypes[tc.op])
			got := prototext.Format(&sbom.NodeList{Nodes: selected})
			want := prototext.Format(&sbom.NodeList{Nodes: tc.expect})
			require.Equal(t, want, got)
		})
	}
}

func TestIngestRepo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		sampleDir string // A directory in testdata/ containing the
		expected  *sbom.NodeList
	}{{
		name:      "simple Python",
		sampleDir: "simple-python",
		expected: &sbom.NodeList{
			Nodes: []*sbom.Node{{
				Type:    sbom.Node_PACKAGE,
				Name:    "PyYAML",
				Version: "5.3.1",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:pypi/pyyaml@5.3.1",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "requirements.txt",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "stevedore",
				Version: "1.20.0",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:pypi/stevedore@1.20.0",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "requirements.txt",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "colorama",
				Version: "0.3.9",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:pypi/colorama@0.3.9",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "requirements.txt",
				}},
			}},
			// Can't pick up "rich" without a version
		},
	}, {
		name:      "simple JavaScript",
		sampleDir: "simple-js", // From MrRio/vtop
		expected: &sbom.NodeList{
			Nodes: []*sbom.Node{{
				// We _can_ detect the top-level package and version in JavaScript, unlike Python
				Type:    sbom.Node_PACKAGE,
				Name:    "vtop",
				Version: "0.6.1",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/vtop@0.6.1",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "package.json",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "balanced-match",
				Version: "1.0.0",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/balanced-match@1.0.0",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "blessed",
				Version: "0.1.81",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/blessed@0.1.81",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "brace-expansion",
				Version: "1.1.11",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/brace-expansion@1.1.11",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "commander",
				Version: "2.11.0",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/commander@2.11.0",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "concat-map",
				Version: "0.0.1",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/concat-map@0.0.1",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "drawille",
				Version: "1.1.0",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/drawille@1.1.0",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "fs.realpath",
				Version: "1.0.0",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/fs.realpath@1.0.0",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "glob",
				Version: "7.1.2",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/glob@7.1.2",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "husky:", // TODO: This is probably a bug in scalibr!
				Version: "0.11.9",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/husky%3A@0.11.9",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "inflight",
				Version: "1.0.6",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/inflight@1.0.6",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "inherits",
				Version: "2.0.3",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/inherits@2.0.3",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "inpath",
				Version: "1.0.2",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/inpath@1.0.2",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "is-ci",
				Version: "1.0.9",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/is-ci@1.0.9",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "minimatch",
				Version: "3.0.4",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/minimatch@3.0.4",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "mute-stream",
				Version: "0.0.6",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/mute-stream@0.0.6",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "normalize-path",
				Version: "1.0.0",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/normalize-path@1.0.0",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "once",
				Version: "1.4.0",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/once@1.4.0",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "os-utils",
				Version: "0.0.14",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/os-utils@0.0.14",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "path-is-absolute",
				Version: "1.0.1",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/path-is-absolute@1.0.1",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "pidof",
				Version: "1.0.2",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/pidof@1.0.2",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "read",
				Version: "1.0.7",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/read@1.0.7",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "sudo",
				Version: "1.0.3",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/sudo@1.0.3",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "use-strict:",
				Version: "1.0.1",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/use-strict%3A@1.0.1",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}, {
				Type:    sbom.Node_PACKAGE,
				Name:    "wrappy",
				Version: "1.0.2",
				Identifiers: map[int32]string{
					int32(sbom.SoftwareIdentifierType_PURL): "pkg:npm/wrappy@1.0.2",
				},
				Properties: []*sbom.Property{{
					Name: "sourceFile",
					Data: "yarn.lock",
				}},
			}},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url := "https://some.url/repo"
			branch := "main"
			repoPb := &v1.Repository{
				CloneUrl: url,
			}
			cfg := map[string]any{
				"repo": map[string]string{
					"branch": branch,
				},
			}

			fs := osfs.New(filepath.Join("testdata", tc.sampleDir))

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)
			ctx := context.Background()

			gitStore := memory.NewStorage()
			require.NoError(t, gitStore.SetReference(plumbing.NewHashReference(plumbing.Main, plumbing.ZeroHash)))
			repo, err := git.InitWithOptions(gitStore, fs, git.InitOptions{DefaultBranch: plumbing.Main})
			require.NoError(t, err)

			gitProv := mock_github.NewMockGit(ctrl)
			gitProv.EXPECT().Clone(gomock.Any(), url, branch).Return(repo, nil)

			gi, err := NewDepsIngester(nil, gitProv)
			require.NoError(t, err)

			result, err := gi.Ingest(ctx, repoPb, cfg)
			require.NoError(t, err)
			nodes := result.Object.(map[string]any)["node_list"].(*sbom.NodeList)

			diff := cmp.Diff(tc.expected.Nodes, nodes.Nodes,
				cmpopts.SortSlices(func(a, b *sbom.Node) bool {
					return nodeSorter(a, b) < 0
				}),
				cmp.Transformer("IgnoreId", func(n *sbom.Node) *sbom.Node {
					n.Id = ""
					return n
				}),
			)
			if diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}
