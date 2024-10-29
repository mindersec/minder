// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/structpb"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	workflowWithPullRequestTargetObject = `on:
  pull_request_target:
    branches:
      - main
  push:
    branches:
      - main
`
	workflowWithPullRequestTargetSeq = `on:
  - pull_request_target
  - push
`
	workflowWithPullRequestTargetDirect = `on:
  pull_request_target
`

	workflowNoPullRequestTargetObject = `on:
  push:
    branches:
      - main
`

	workflowNoPullRequestTargetSeq = `on:
  - push
`

	workflowNoPullRequestTargetDirect = `on: push
`

	workflowDispatch = `on: workflow_dispatch
`
	workflowPullRequestTargetObjectFileName   = ".github/workflows/prt_object.yaml"
	workflowPullRequestTargetSeqFileName      = ".github/workflows/prt_seq.yaml"
	workflowPullRequestTargetDirectFileName   = ".github/workflows/prt_direct.yaml"
	workflowNoPullRequestTargetObjectFileName = ".github/workflows/nprt_object.yaml"
	workflowNoPullRequestTargetSeqFileName    = ".github/workflows/nprt_seq.yaml"
	workflowNoPullRequestTargetDirectFileName = ".github/workflows/nprt_direct.yaml"

	readmeOutsideContent           = "readme outside workflows"
	nonMatchingFileNameOutsideTree = "README.md"
	readmeInsideContent            = "readme inside workflows"
	nonMatchingFileNameInsideTree  = ".github/workflows/README.txt"
)

type testFileContents struct {
	origContents     string
	modifiedContents string
}

type testFileMap map[string]testFileContents

type fsConstructorOpt func(*testing.T, billy.Filesystem)

func newTestFS(t *testing.T, opts ...fsConstructorOpt) billy.Filesystem {
	t.Helper()
	fs := memfs.New()
	for _, opt := range opts {
		opt(t, fs)
	}
	return fs
}

func withFile(path, contents string) fsConstructorOpt {
	return func(t *testing.T, fs billy.Filesystem) {
		t.Helper()
		dir := filepath.Dir(path)
		if dir != "." {
			err := fs.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("failed to create directory %s: %v", dir, err)
			}
		}

		f, err := fs.Create(path)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
		defer f.Close()

		_, err = f.Write([]byte(contents))
		if err != nil {
			t.Fatalf("failed to write to file %s: %v", path, err)
		}
	}
}

type modificationConstructorOpt func(*modificationConstructorParams)

func newModificationParams(opts ...modificationConstructorOpt) *modificationConstructorParams {
	params := &modificationConstructorParams{}
	for _, opt := range opts {
		opt(params)
	}
	return params
}

func withParams(m map[string]any) modificationConstructorOpt {
	return func(p *modificationConstructorParams) {
		s, err := structpb.NewStruct(m)
		if err != nil {
			panic(fmt.Sprintf("failed to create struct from map: %v", err))
		}
		p.prCfg = &minderv1.RuleType_Definition_Remediate_PullRequestRemediation{
			Params: s,
		}
	}
}

func TestYQExecute(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name         string
		yqParams     func(*testing.T) *modificationConstructorParams
		testFiles    map[string]testFileContents
		fs           func(*testing.T, testFileMap) billy.Filesystem
		checkEntries func(*testing.T, testFileMap, []*fsEntry)
		createErr    string
	}{
		{
			name: "TestYQExecute",
			yqParams: func(t *testing.T) *modificationConstructorParams {
				t.Helper()

				return newModificationParams(withParams(map[string]any{
					"expression": `
.on |= (
  select(type == "!!map") | with_entries(select(.key != "pull_request_target"))
) |
.on |= (
  select(type == "!!seq") | map(select(. != "pull_request_target"))
) |
(.on | select(. == "pull_request_target")) = "workflow_dispatch" |
(.on |= (select(length > 0) // "workflow_dispatch"))
`,
					"patterns": []any{
						map[string]any{
							"pattern": ".github/workflows/*.yml",
							"type":    "glob",
						},
						map[string]any{
							"pattern": ".github/workflows/*.yaml",
							"type":    "glob",
						},
					},
				}))
			},
			testFiles: testFileMap{
				workflowPullRequestTargetObjectFileName: {
					origContents:     workflowWithPullRequestTargetObject,
					modifiedContents: workflowNoPullRequestTargetObject,
				},
				workflowPullRequestTargetSeqFileName: {
					origContents:     workflowWithPullRequestTargetSeq,
					modifiedContents: workflowNoPullRequestTargetSeq,
				},
				workflowPullRequestTargetDirectFileName: {
					origContents:     workflowWithPullRequestTargetDirect,
					modifiedContents: workflowDispatch,
				},
				workflowNoPullRequestTargetObjectFileName: {
					origContents:     workflowNoPullRequestTargetObject,
					modifiedContents: workflowNoPullRequestTargetObject,
				},
				workflowNoPullRequestTargetSeqFileName: {
					origContents:     workflowNoPullRequestTargetSeq,
					modifiedContents: workflowNoPullRequestTargetSeq,
				},
				workflowNoPullRequestTargetDirectFileName: {
					origContents:     workflowNoPullRequestTargetDirect,
					modifiedContents: workflowNoPullRequestTargetDirect,
				},
				nonMatchingFileNameInsideTree: {
					origContents:     readmeInsideContent,
					modifiedContents: readmeInsideContent,
				},
				nonMatchingFileNameOutsideTree: {
					origContents:     readmeOutsideContent,
					modifiedContents: readmeOutsideContent,
				},
			},
			fs: func(t *testing.T, testFiles testFileMap) billy.Filesystem {
				t.Helper()
				opts := make([]fsConstructorOpt, 0, len(testFiles))
				for path, contents := range testFiles {
					opts = append(opts, withFile(path, contents.origContents))
				}
				return newTestFS(t, opts...)
			},
			checkEntries: func(t *testing.T, testFiles testFileMap, entries []*fsEntry) {
				t.Helper()

				// two non-matching files
				require.Len(t, entries, len(testFiles)-2)
				testFilesCopy := maps.Clone(testFiles)
				for _, entry := range entries {
					if entry.Path == nonMatchingFileNameOutsideTree || entry.Path == nonMatchingFileNameInsideTree {
						t.Errorf("matched file %s that shouldn't be matched", entry.Path)
					}
					file, ok := testFilesCopy[entry.Path]
					require.True(t, ok)
					delete(testFilesCopy, entry.Path) // remove the entry from the map to make sure we catch duplicates
					if file.modifiedContents != "" {
						require.Equal(t, file.modifiedContents, entry.Content)
					}
				}
			},
		},
		{
			// this won't do anything but won't crash
			name: "No config",
			yqParams: func(t *testing.T) *modificationConstructorParams {
				t.Helper()
				return newModificationParams()
			},
			testFiles: testFileMap{},
			fs: func(t *testing.T, _ testFileMap) billy.Filesystem {
				t.Helper()
				return newTestFS(t)
			},
			checkEntries: func(t *testing.T, _ testFileMap, entries []*fsEntry) {
				t.Helper()
				require.Len(t, entries, 0)
			},
		},
		{
			name: "No matching files",
			yqParams: func(t *testing.T) *modificationConstructorParams {
				t.Helper()

				return newModificationParams(withParams(map[string]any{
					"expression": `
(.on |= (select(length > 0) // "workflow_dispatch"))
`,
					"patterns": []any{
						map[string]any{
							"pattern": ".github/workflows/*.yml",
							"type":    "glob",
						},
						map[string]any{
							"pattern": ".github/workflows/*.yaml",
							"type":    "glob",
						},
					},
				}))
			},
			testFiles: testFileMap{
				nonMatchingFileNameInsideTree: {
					origContents:     readmeInsideContent,
					modifiedContents: readmeInsideContent,
				},
				nonMatchingFileNameOutsideTree: {
					origContents:     readmeOutsideContent,
					modifiedContents: readmeOutsideContent,
				},
			},
			fs: func(t *testing.T, testFiles testFileMap) billy.Filesystem {
				t.Helper()
				opts := make([]fsConstructorOpt, 0, len(testFiles))
				for path, contents := range testFiles {
					opts = append(opts, withFile(path, contents.origContents))
				}
				return newTestFS(t, opts...)
			},
			checkEntries: func(t *testing.T, _ testFileMap, entries []*fsEntry) {
				t.Helper()
				require.Len(t, entries, 0)
			},
		},
		{
			name: "Bad expression",
			yqParams: func(t *testing.T) *modificationConstructorParams {
				t.Helper()

				return newModificationParams(withParams(map[string]any{
					"expression": `
(.on |= if oops then spoo)
`,
					"patterns": []any{
						map[string]any{
							"pattern": ".github/workflows/*.yml",
							"type":    "glob",
						},
						map[string]any{
							"pattern": ".github/workflows/*.yaml",
							"type":    "glob",
						},
					},
				}))
			},
			testFiles: testFileMap{
				workflowPullRequestTargetObjectFileName: {
					origContents:     workflowWithPullRequestTargetObject,
					modifiedContents: workflowNoPullRequestTargetObject,
				},
			},
			fs: func(t *testing.T, testFiles testFileMap) billy.Filesystem {
				t.Helper()
				opts := make([]fsConstructorOpt, 0, len(testFiles))
				for path, contents := range testFiles {
					opts = append(opts, withFile(path, contents.origContents))
				}
				return newTestFS(t, opts...)
			},
			checkEntries: func(t *testing.T, _ testFileMap, entries []*fsEntry) {
				t.Helper()
				require.Len(t, entries, 0)
			},
			createErr: "cannot execute yq: cannot parse expression",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			params := scenario.yqParams(t)
			fs := scenario.fs(t, scenario.testFiles)
			params.bfs = fs

			yqe, err := newYqExecute(params)
			require.NoError(t, err)
			require.NotNil(t, yqe)

			err = yqe.createFsModEntries(context.Background(), nil)
			if scenario.createErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), scenario.createErr)
			} else {
				require.NoError(t, err)
			}

			entries, err := yqe.modifyFs()
			require.NoError(t, err)
			scenario.checkEntries(t, scenario.testFiles, entries)

		})
	}
}
