// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/engine/interfaces"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	engif "github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
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

func TestYQExactPattern(t *testing.T) {
	t.Parallel()

	const filename = ".github/config/settings.yaml"
	const expression = `.enabled = true`
	for _, tt := range []struct {
		name       string
		pattern    string
		typeName   string
		create     bool
		original   string
		expression string
		readErr    error
		want       string
		wantErr    string
	}{
		{name: "create nested file", pattern: filename, create: true, want: "enabled: true\n"},
		{name: "preserve comments", pattern: filename, create: true,
			original: "# configuration\nenabled: false # keep this\nother: value\n",
			want:     "# configuration\nenabled: true # keep this\nother: value\n"},
		{name: "update without creation", pattern: filename, original: "enabled: false\n", want: "enabled: true\n"},
		{name: "missing without creation", pattern: filename, wantErr: "cannot read file"},
		{name: "permission error", pattern: filename, create: true, readErr: os.ErrPermission,
			wantErr: "cannot read file"},
		{name: "wrapped not exist", pattern: filename, create: true,
			readErr: fmt.Errorf("open: %w", os.ErrNotExist), want: "enabled: true\n"},
		{name: "literal metacharacters", pattern: "config/[settings].yaml", create: true, want: "enabled: true\n"},
		{name: "glob does not create", pattern: ".github/config/*.yaml", typeName: "glob", create: true},
		{name: "invalid expression does not create", pattern: filename, create: true,
			expression: ".enabled = [", wantErr: "cannot parse expression"},
		{name: "invalid YAML is not overwritten", pattern: filename, create: true,
			original: "enabled: [", wantErr: "cannot evaluate expression"},
		{name: "empty path", create: true, wantErr: "invalid exact path"},
		{name: "root path", pattern: ".", create: true, wantErr: "invalid exact path"},
		{name: "absolute path", pattern: "/settings.yaml", create: true, wantErr: "invalid exact path"},
		{name: "parent path", pattern: "../settings.yaml", create: true, wantErr: "invalid exact path"},
		{name: "nested parent", pattern: "config/../settings.yaml", create: true, wantErr: "invalid exact path"},
		{name: "trailing slash", pattern: "config/", create: true, wantErr: "invalid exact path"},
		{name: "windows path", pattern: `C:\settings.yaml`, create: true, wantErr: "invalid exact path"},
		{name: "windows slash path", pattern: "C:/settings.yaml", create: true, wantErr: "invalid exact path"},
		{name: "backslash parent", pattern: `..\settings.yaml`, create: true, wantErr: "invalid exact path"},
		{name: "NUL path", pattern: "settings\x00.yaml", create: true, wantErr: "invalid exact path"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := newTestFS(t)
			if tt.original != "" {
				withFile(tt.pattern, tt.original)(t, fs)
			}
			typeName := tt.typeName
			if typeName == "" {
				typeName = "exact"
			}
			expr := tt.expression
			if expr == "" {
				expr = expression
			}
			pattern := map[string]any{"type": typeName, "pattern": tt.pattern}
			if tt.create {
				pattern["create_if_needed"] = true
			}
			params := newModificationParams(withParams(map[string]any{
				"expression": expr,
				"patterns":   []any{pattern},
			}))
			params.bfs = fs
			if tt.readErr != nil {
				params.bfs = &yqReadErrorFS{Filesystem: fs, err: tt.readErr}
			}
			modifier, err := newYqExecute(params)
			require.NoError(t, err)
			actionParams := &interfaces.EvalStatusParams{Rule: &models.RuleInstance{}}
			err = modifier.createFsModEntries(context.Background(), nil, actionParams)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				if tt.readErr != nil {
					require.ErrorIs(t, err, tt.readErr)
				}
			} else {
				require.NoError(t, err)
			}
			// Preparing entries (including failures) must not change the ingest filesystem.
			files, err := fs.ReadDir(".")
			require.NoError(t, err)
			if tt.original == "" {
				require.Empty(t, files)
			} else {
				require.Equal(t, tt.original, readYQTestFile(t, fs, tt.pattern))
			}
			if tt.wantErr != "" {
				return
			}
			entries, err := modifier.modifyFs()
			require.NoError(t, err)
			if tt.want == "" {
				require.Empty(t, entries)
				return
			}
			require.Len(t, entries, 1)
			require.Equal(t, tt.pattern, entries[0].Path)
			require.Equal(t, tt.want, readYQTestFile(t, fs, tt.pattern))

			// A fresh application to the resulting file must preserve the same bytes.
			params.bfs = fs
			modifier, err = newYqExecute(params)
			require.NoError(t, err)
			require.NoError(t, modifier.createFsModEntries(context.Background(), nil, actionParams))
			_, err = modifier.modifyFs()
			require.NoError(t, err)
			require.Equal(t, tt.want, readYQTestFile(t, fs, tt.pattern))
		})
	}
}

type yqReadErrorFS struct {
	billy.Filesystem
	err error
}

func (fs *yqReadErrorFS) Open(string) (billy.File, error) {
	return nil, fs.err
}

func readYQTestFile(t *testing.T, fs billy.Filesystem, path string) string {
	t.Helper()
	file, err := fs.Open(path)
	require.NoError(t, err)
	defer file.Close()
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	return string(content)
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
		evalParams   map[string]any
		evalOutput   any
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
					normalizedPath := filepath.ToSlash(entry.Path)
					if normalizedPath == nonMatchingFileNameOutsideTree || normalizedPath == nonMatchingFileNameInsideTree {
						t.Errorf("matched file %s that shouldn't be matched", normalizedPath)
					}
					file, ok := testFilesCopy[normalizedPath]
					require.True(t, ok)
					delete(testFilesCopy, normalizedPath) // remove the entry from the map to make sure we catch duplicates
					if file.modifiedContents != "" {
						require.Equal(t, file.modifiedContents, entry.Content)
					}
				}
			},
		},
		{
			name: "TestTemplatedExpression",
			yqParams: func(t *testing.T) *modificationConstructorParams {
				t.Helper()

				return newModificationParams(withParams(map[string]any{
					"expression": `
.updates += [
{{- $last := len (slice .EvalResultOutput.Ecosystems 1) -}}
{{- $schedule := .Params.schedule -}}
{{- range $index, $item := .EvalResultOutput.Ecosystems -}}
{"package-ecosystem": "{{ $item.name }}",
	{{- if index $item "directory" }} "directory": "{{ $item.directory }}", {{- end }}
	"schedule": {"interval": "{{ $schedule }}"}}
	{{- /* YQ is sensitive to trailing commas, so use last to omit them */ -}}
	{{- if lt $index $last }},{{ end }}
{{- end -}}
]
`,
					"patterns": []any{
						map[string]any{
							"pattern": ".github/dependabot.yml",
							"type":    "glob",
						},
						map[string]any{
							"pattern": ".github/dependabot.yaml",
							"type":    "glob",
						},
					},
				}))
			},
			evalParams: map[string]any{
				"schedule": "weekly",
			},
			evalOutput: map[string]any{
				"Ecosystems": []any{
					map[string]any{"name": "gomod", "directory": "/go"},
					map[string]any{"name": "github-actions"},
				},
			},
			testFiles: testFileMap{
				".github/dependabot.yml": {
					origContents: `# Dependabot configuration file
version: 2
updates:
  - package-ecosystem: "npm"
    directory: "/docs"
  - package-ecosystem: npm
    directory: /frontend
    schedule:
      interval: weekly
`,
					modifiedContents: `# Dependabot configuration file
version: 2
updates:
  - package-ecosystem: "npm"
    directory: "/docs"
  - package-ecosystem: npm
    directory: /frontend
    schedule:
      interval: weekly
  - package-ecosystem: gomod
    directory: /go
    schedule:
      interval: weekly
  - package-ecosystem: github-actions
    schedule:
      interval: weekly
`,
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
				require.Len(t, entries, len(testFiles))
				testFilesCopy := maps.Clone(testFiles)
				for _, entry := range entries {
					normalizedPath := filepath.ToSlash(entry.Path)
					file, ok := testFilesCopy[normalizedPath]
					require.True(t, ok)
					delete(testFilesCopy, normalizedPath) // remove the entry from the map to make sure we catch duplicates
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

			ifParams := interfaces.EvalStatusParams{
				Rule: &models.RuleInstance{
					Params: scenario.evalParams,
				},
			}
			if scenario.evalOutput != nil {
				ifParams.SetEvalResult(&engif.EvaluationResult{
					Output: scenario.evalOutput,
				})
			}

			err = yqe.createFsModEntries(context.Background(), nil, &ifParams)
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
