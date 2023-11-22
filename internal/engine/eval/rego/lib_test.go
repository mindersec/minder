// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package rego_test

import (
	"context"
	"testing"

	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/require"

	engerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/rego"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestFileExistsWithExistingFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	// Create a file
	_, err := fs.Create("foo")
	require.NoError(t, err, "could not create file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	file.exists("foo")
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileExistsWithNonExistentFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	file.exists("unexistent")
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "could not evaluate")
}

func TestFileReadWithContentsMatching(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	// Create a file
	f, err := fs.Create("foo")
	require.NoError(t, err, "could not create file")

	_, err = f.Write([]byte("bar"))
	require.NoError(t, err, "could not write to file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	contents := file.read("foo")
	contents == "bar"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileReadWithContentsNotMatching(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	// Create a file
	f, err := fs.Create("foo")
	require.NoError(t, err, "could not create file")

	_, err = f.Write([]byte("baz"))
	require.NoError(t, err, "could not write to file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	contents := file.read("foo")
	contents == "bar"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "could not evaluate")
}

func TestFileLsWithUnexistentFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls("unexistent")
	is_null(files)
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileLsWithEmptyDirectory(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	err := fs.MkdirAll("foo", 0755)
	require.NoError(t, err, "could not create directory")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls("foo")
	count(files) == 0
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileLsWithSingleFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	err := fs.MkdirAll("foo", 0755)
	require.NoError(t, err, "could not create directory")

	// Create a file
	_, err = fs.Create("foo/bar")
	require.NoError(t, err, "could not create file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls("foo")
	count(files) == 1
	files[0] == "foo/bar"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileLsWithSingleFileDirect(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	err := fs.MkdirAll("foo", 0755)
	require.NoError(t, err, "could not create directory")

	// Create a file
	_, err = fs.Create("foo/bar")
	require.NoError(t, err, "could not create file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls("foo/bar")
	count(files) == 1
	files[0] == "foo/bar"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileLsWithMultipleFiles(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	err := fs.MkdirAll("foo", 0755)
	require.NoError(t, err, "could not create directory")

	// Create a files
	_, err = fs.Create("foo/bar")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("foo/baz")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("foo/bat")
	require.NoError(t, err, "could not create file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls("foo")
	count(files) == 3
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileLsWithSimpleSymlink(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	err := fs.MkdirAll("foo", 0755)
	require.NoError(t, err, "could not create directory")

	// Create a files
	_, err = fs.Create("foo/bar")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("foo/baz")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("foo/bat")
	require.NoError(t, err, "could not create file")

	err = fs.Symlink("foo", "beer")
	require.NoError(t, err, "could not create symlink")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls("beer")
	count(files) == 3
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileLsWithSymlinkToDir(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	err := fs.MkdirAll("foo", 0755)
	require.NoError(t, err, "could not create directory")

	// Create a files
	_, err = fs.Create("foo/bar")
	require.NoError(t, err, "could not create file")
	err = fs.Symlink("foo/bar", "foo/baz")
	require.NoError(t, err, "could not create file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls("foo/baz")
	count(files) == 1
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

const (
	buildWorkflow = `
on:
  workflow_call:
jobs:
  build:
    name: Verify build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4 # v3.5.0
      - name: Extract version of Go to use
        run: echo "GOVERSION=$(sed -n 's/^go \([0-9.]*\)/\1/p' go.mod)" >> $GITHUB_ENV
      - uses: actions/setup-go@v4 # v4.0.0
        with:
          go-version-file: 'go.mod'
      - name: build
        run: make build
`

	// checkout is missing from this workflow on purpose - I (jakub) wanted
	// to test that the actions are merged together
	testWorkflow = `
on:
  workflow_call:
jobs:
  test:
    name: Unit testing
    runs-on: ubuntu-latest
    steps:
      # Install Go on the VM running the action.
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version-file: 'go.mod'

      - name: Set up helm (test dependency)
        uses: azure/setup-helm@v3

      # Install gotestfmt on the VM running the action.
      - name: Set up gotestfmt
        uses: GoTestTools/gotestfmt-action@v2
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      # copy config file into place
      - name: Copy config file
        run: cp config/config.yaml.example ./config.yaml

      # Run the tests
      - name: Run tests
        run: make test
`
)

func TestListGithubActionsDirectory(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	err := fs.MkdirAll("workflows", 0755)
	require.NoError(t, err, "could not create directory")

	buildFile, err := fs.Create("workflows/build.yml")
	require.NoError(t, err, "could not create build file")
	_, err = buildFile.Write([]byte(buildWorkflow))
	require.NoError(t, err, "could not write to build file")

	testFile, err := fs.Create("workflows/test.yml")
	require.NoError(t, err, "could not create test file")
	_, err = testFile.Write([]byte(testWorkflow))
	require.NoError(t, err, "could not write to test file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	actions := github_workflow.ls_actions("workflows")
	expected_set = {"actions/checkout", "actions/setup-go", "GoTestTools/gotestfmt-action", "azure/setup-helm"}
	actions == expected_set
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestListGithubActionsFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	buildFile, err := fs.Create("build.yml")
	require.NoError(t, err, "could not create build file")
	_, err = buildFile.Write([]byte(buildWorkflow))
	require.NoError(t, err, "could not write to build file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	actions := github_workflow.ls_actions("build.yml")
	expected_set = {"actions/checkout", "actions/setup-go"}
	actions == expected_set
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}
