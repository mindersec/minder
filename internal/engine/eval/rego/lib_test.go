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

	engerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/eval/rego"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

func TestFileExistsWithExistingFile(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	// Create a file
	_, err := fs.Create("foo")
	require.NoError(t, err, "could not create file")

	e, err := rego.NewRegoEvaluator(
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

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
