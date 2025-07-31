// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	memfs "github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"

	engerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/rego"
	"github.com/mindersec/minder/internal/engine/options"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/flags"
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
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileExistsInBase(t *testing.T) {
	t.Parallel()
	fs := memfs.New()

	_, err := fs.Create("foo")
	require.NoError(t, err, "could not create file")

	featureClient := &flags.FakeClient{}
	featureClient.Data = map[string]any{"git_pr_diffs": true}
	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
    base_file.exists("foo")
}`,
		},
		options.WithFlagsClient(featureClient),
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		BaseFs: fs,
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "could not evaluate")
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "could not evaluate")
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
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

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestListYamlUsingLSGlob(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	require.NoError(t, fs.MkdirAll(".github", 0755))

	_, err := fs.Create(".github/dependabot.yaml")
	require.NoError(t, err, "could not create dependabot file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls_glob(".github/dependabot.y*ml")
	count(files) == 1
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestListYamlsUsingLSGlob(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	require.NoError(t, fs.MkdirAll(".github", 0755))
	require.NoError(t, fs.MkdirAll(".github/workflows", 0755))

	_, err := fs.Create(".github/workflows/security.yaml")
	require.NoError(t, err, "could not create sec workflow file")

	_, err = fs.Create(".github/workflows/build.yml")
	require.NoError(t, err, "could not create build workflow file")

	_, err = fs.Create(".github/workflows/release.yaml")
	require.NoError(t, err, "could not create release workflow file")

	// non-matching file
	_, err = fs.Create(".github/workflows/README.md")
	require.NoError(t, err, "could not create README file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.ls_glob(".github/workflows/*.y*ml")
	count(files) == 3
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestHTTPTypeWithTextFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	txtfile, err := fs.Create("textfile")
	require.NoError(t, err, "could not create sec workflow file")

	_, err = txtfile.Write([]byte("foo"))
	require.NoError(t, err, "could not write to file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	htype := file.http_type("textfile")
	htype == "text/plain; charset=utf-8"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestHTTPTypeWithBinaryFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	binfile, err := fs.Create("binfile")
	require.NoError(t, err, "could not create sec workflow file")

	// write binary file
	_, err = binfile.Write([]byte{0x00, 0x01, 0x02, 0x03})
	require.NoError(t, err, "could not write to file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	htype := file.http_type("binfile")
	htype == "application/octet-stream"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileWalk(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	require.NoError(t, fs.MkdirAll("foo", 0755), "could not create directory")
	require.NoError(t, fs.MkdirAll("bar", 0755), "could not create directory")

	// Create a files
	_, err := fs.Create("foo/bar")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("foo/baz")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("foo/bat")
	require.NoError(t, err, "could not create file")

	_, err = fs.Create("bar/bar")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("bar/baz")
	require.NoError(t, err, "could not create file")

	_, err = fs.Create("beer")
	require.NoError(t, err, "could not create file")
	_, err = fs.Create("hmmm")
	require.NoError(t, err, "could not create file")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	files := file.walk(".")
	count(files) == 7
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestFileArchive(t *testing.T) {
	t.Parallel()
	fs := memfs.New()
	require.NoError(t, fs.MkdirAll("foo", 0755), "could not create directory")
	require.NoError(t, fs.MkdirAll("bar", 0755), "could not create directory")

	require.NoError(t, billyutil.WriteFile(fs, "foo/bar", []byte("bar"), 0644))
	require.NoError(t, billyutil.WriteFile(fs, "foo/baz", []byte("bar"), 0644))
	require.NoError(t, billyutil.WriteFile(fs, "file.txt", []byte("words"), 0644))
	require.NoError(t, billyutil.WriteFile(fs, "README", []byte("docs"), 0644))

	// N.B. This was constructed by examining the output of the tarball, and
	// and verifying by untarring the data with `cat file | tar -tzvf -`
	expectedTarball := []byte{
		31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 236, 147, 65, 10, 2, 49, 12, 69, 115,
		148, 57, 129, 254, 98, 211, 158, 103, 68, 11, 130, 16, 232, 84, 20, 79,
		47, 234, 202, 34, 10, 74, 170, 67, 243, 54, 153, 213, 252, 52, 159, 151,
		68, 72, 27, 0, 136, 204, 183, 9, 160, 158, 79, 190, 99, 116, 142, 6, 86,
		223, 140, 136, 14, 83, 25, 51, 1, 223, 254, 167, 126, 220, 76, 72, 34,
		203, 245, 152, 85, 51, 174, 247, 8, 222, 191, 232, 127, 245, 216, 191,
		131, 103, 208, 208, 228, 136, 157, 247, 175, 221, 189, 241, 223, 220,
		253, 63, 171, 102, 124, 226, 127, 48, 255, 155, 96, 254, 247, 77, 218,
		237, 183, 139, 114, 42, 154, 25, 239, 253, 231, 218, 255, 96, 254, 183,
		225, 40, 121, 51, 253, 122, 9, 195, 48, 12, 163, 57, 151, 0, 0, 0, 255,
		255, 203, 184, 208, 59, 0, 18, 0, 0,
	}

	featureClient := &flags.FakeClient{}
	featureClient.Data = map[string]any{"tar_gz_functions": true}
	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder
import rego.v1

tarball := file.archive(["foo", "file.txt"])
encoded := base64.encode(tarball)
expectedTar := base64.decode(input.profile.expected)
violations contains {"msg": sprintf("Expected: %s", [input.profile.expected])} if tarball != expectedTar
violations contains {"msg": sprintf("Got     : %s", [encoded])} if tarball != expectedTar
`,
		},
		options.WithFlagsClient(featureClient),
	)
	require.NoError(t, err, "could not create evaluator")

	policy := map[string]any{
		// Encode to string in Go, to force checking that Go & Rego perform the same encoding
		"expected": base64.StdEncoding.EncodeToString(expectedTarball),
	}

	_, err = e.Eval(context.Background(), policy, nil, &interfaces.Ingested{
		Object: nil,
		Fs:     fs,
	})
	require.NoError(t, err, "could not evaluate")
}

func TestJQIsTrue(t *testing.T) {
	t.Parallel()

	scenario := []struct {
		name    string
		yaml    string
		matches bool
	}{
		{
			name: "match a string",
			yaml: `
on:
  pull_request_target
`,
			matches: true,
		},
		{
			name: "don't match a different string",
			yaml: `
on:
  push
`,
			matches: false,
		},
		{
			name: "match an array",
			yaml: `
on:
  - pull_request_target
`,
			matches: true,
		},
		{
			name: "don't match an array without pull_request_target",
			yaml: `
on:
  - push
`,
			matches: false,
		},
		{
			name: "match an array with multiple elements",
			yaml: `
on:
  - pull_request_target
  - push
`,
			matches: true,
		},
		{
			name: "don't match an array with multiple elements without pull_request_target",
			yaml: `
on:
  - push
  - workflow_dispatch
`,
			matches: false,
		},
		{
			name: "match an object",
			yaml: `
on:
  pull_request_target:
    types: [opened, synchronize]
`,
			matches: true,
		},
		{
			name: "don't match an object without pull_request_target",
			yaml: `
on:
  push:
    branches: [main]
`,
			matches: false,
		},
		{
			name: "match a complex object",
			yaml: `
on:
  push:
    branches: [master]
  pull_request_target:
    types: [opened, synchronize]
`,
			matches: true,
		},
		{
			name: "don't match a complex object without pull_request_target",
			yaml: `
on:
  push:
    branches: [master]
  workflow_dispatch:
    inputs:
      logLevel:
        description: 'Log level'
        required: true
        default: 'warning'
`,
			matches: false,
		},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			const jqQuery = `.on | (type == "string" and . == "pull_request_target") or (type == "object" and has("pull_request_target")) or (type == "array" and any(.[]; . == "pull_request_target"))`

			fs := memfs.New()

			// Create a unique file name for each test
			workflowFile := fmt.Sprintf("workflow_%d.yaml", time.Now().UnixNano())
			f, err := fs.Create(workflowFile)
			require.NoError(t, err, "could not create file in memfs")

			_, err = f.Write([]byte(s.yaml))
			require.NoError(t, err, "could not write to file in memfs")
			err = f.Close()
			require.NoError(t, err, "could not close file in memfs")

			regoCode := fmt.Sprintf(`
package minder

default allow = false

allow {
	workflowstr := file.read("%s")
    parsed := parse_yaml(workflowstr)
	jq.is_true(parsed, %q)
}`, workflowFile, jqQuery)

			e, err := rego.NewRegoEvaluator(
				&minderv1.RuleType_Definition_Eval_Rego{
					Type: rego.DenyByDefaultEvaluationType.String(),
					Def:  regoCode,
				},
			)
			require.NoError(t, err, "could not create evaluator")

			emptyPol := map[string]any{}

			var evalErr *engerrors.EvaluationError
			_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
				Object: nil,
				Fs:     fs,
			})
			if s.matches {
				require.NoError(t, err, "expected the policy to be allowed")
			} else if !errors.As(err, &evalErr) {
				t.Fatalf("expected the policy to be denied by default, got: %v", err)
			}
		})
	}
}

func TestParseYaml(t *testing.T) {
	t.Parallel()

	scenario := []struct {
		name    string
		yaml    string
		want    string
		wantErr bool
	}{
		{
			name: "simple key-value",
			yaml: "foo: bar",
			want: `{"foo": "bar"}`,
		},
		{
			name: "nested structure",
			yaml: `
foo:
  bar:
    baz: qux`,
			want: `{"foo": {"bar": {"baz": "qux"}}}`,
		},
		{
			name: "yaml with 'on' key",
			yaml: `
on: push
name: test`,
			want: `{"on": "push", "name": "test"}`,
		},
		{
			name: "complex github workflow",
			yaml: `
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]`,
			want: `{"on": {"push": {"branches": ["main"]}, "pull_request": {"branches": ["main"]}}}`,
		},
		{
			name: "array values",
			yaml: `
items:
  - foo
  - bar
  - baz`,
			want: `{"items": ["foo", "bar", "baz"]}`,
		},
		{
			name: "mixed types",
			yaml: `
string: hello
number: 42
boolean: true
null_value: null
array: [1, 2, 3]`,
			want: `{"string": "hello", "number": 42, "boolean": true, "null_value": null, "array": [1, 2, 3]}`,
		},
		{
			name:    "invalid yaml",
			yaml:    "foo: [bar: invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			regoCode := fmt.Sprintf(`
package minder

default allow = false

allow {
    parsed := parse_yaml(%q)
    expected := json.unmarshal(%q)
    parsed == expected
}`, s.yaml, s.want)

			e, err := rego.NewRegoEvaluator(
				&minderv1.RuleType_Definition_Eval_Rego{
					Type: rego.DenyByDefaultEvaluationType.String(),
					Def:  regoCode,
				},
			)
			require.NoError(t, err, "could not create evaluator")

			_, err = e.Eval(context.Background(), map[string]any{}, nil, &interfaces.Ingested{})

			if s.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseToml(t *testing.T) {
	t.Parallel()

	scenario := []struct {
		name    string
		toml    string
		want    string
		wantErr bool
	}{
		{
			name: "simple key-value",
			toml: "foo = \"bar\"",
			want: `{"foo": "bar"}`,
		},
		{
			name: "nested structure",
			toml: `
[foo.bar]
baz = "qux"`,
			want: `{"foo": {"bar": {"baz": "qux"}}}`,
		},
		{
			name: "array values",
			toml: `
items = ["foo", "bar", "baz"]`,
			want: `{"items": ["foo", "bar", "baz"]}`,
		},
		{
			name: "parse array of tables",
			toml: `
[[items]]
name = "foo"
[[items]]
name = "bar"`,
			want: `{"items": [{"name": "foo"}, {"name": "bar"}]}`,
		},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			regoCode := fmt.Sprintf(`
package minder

default allow = false

allow {
	parsed := parse_toml(%q)
	print(parsed)
	expected := json.unmarshal(%q)
	parsed == expected
}`, s.toml, s.want)

			e, err := rego.NewRegoEvaluator(
				&minderv1.RuleType_Definition_Eval_Rego{
					Type: rego.DenyByDefaultEvaluationType.String(),
					Def:  regoCode,
				},
			)

			require.NoError(t, err, "could not create evaluator")

			_, err = e.Eval(context.Background(), map[string]any{}, nil, &interfaces.Ingested{})

			if s.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExtractDeps(t *testing.T) {
	t.Parallel()

	scenario := []struct {
		name         string
		path         string
		expectedDeps []string
		expectedErr  error
	}{{
		name: "parse all",
		expectedDeps: []string{
			"example.com/othermodule",
			"example.com/thismodule",
			"example.com/thatmodule",
			"stdlib", // Always part of golang binaries.
			"PyYAML",
		},
	}, {
		name: "parse go.mod",
		path: "foo",
		expectedDeps: []string{
			"example.com/othermodule",
			"example.com/thismodule",
			"example.com/thatmodule",
			"stdlib", // Always part of golang binaries.
		},
	}, {
		name: "parse file",
		path: "requirements.txt",
		expectedDeps: []string{
			"PyYAML",
		},
	}, {
		name:        "parse non-existent file",
		path:        "missing",
		expectedErr: engerrors.NewErrEvaluationFailed("denied"),
	}}

	fs := memfs.New()
	require.NoError(t, fs.MkdirAll("foo", 0755), "could not create directory")
	// From https://go.dev/doc/modules/gomod-ref#example
	goMod := `
module example.com/mymodule

go 1.14

require (
    example.com/othermodule v1.2.3
    example.com/thismodule v1.2.3
    example.com/thatmodule v1.2.3
)
`

	require.NoError(t, billyutil.WriteFile(fs, "foo/go.mod", []byte(goMod), 0644))
	require.NoError(t, billyutil.WriteFile(fs, "requirements.txt", []byte("PyYAML>=5.3.1"), 0644))

	featureClient := &flags.FakeClient{}
	featureClient.Data = map[string]any{"dependency_extract": true}
	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			// TODO: update rego for different APIs
			Def: `
package minder
import rego.v1

deps := file.deps(input.profile.path)
depsSet := { x |  x = deps.node_list.nodes[_].name }
expected := { x | x = input.profile.expected[_] }

default allow = false
allow if {
  count(depsSet) > 0
  count(depsSet - expected) == 0
  count(expected - depsSet) == 0
}
`,
		},
		options.WithFlagsClient(featureClient),
	)
	require.NoError(t, err, "could not create evaluator")

	for _, tc := range scenario {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			policy := map[string]any{
				"path":     tc.path,
				"expected": tc.expectedDeps,
			}

			result, err := e.Eval(context.Background(), policy, nil, &interfaces.Ingested{
				Fs: fs,
			})

			if tc.expectedErr == nil {
				t.Logf("Result: %+v", result)
				require.NoError(t, err, "could not evaluate")
			} else {
				require.EqualError(t, err, tc.expectedErr.Error())
			}
			//t.Fail()
		})
	}
}
