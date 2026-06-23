// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ruletest provides a Starlark-based test runner for Minder rule types.
package ruletest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
	"go.starlark.net/syntax"
)

// testCaseRunner is responsible for executing a single Starlark file
// or a single test case. It implements the starlarktest.Reporter interface.
type testCaseRunner struct {
	thread      *starlark.Thread
	fs          fs.FS
	predeclared starlark.StringDict
	failures    []string
}

func (r *Runner) newTestCaseRunner(name string, fileSystem fs.FS) *testCaseRunner {
	tr := &testCaseRunner{
		fs:          fileSystem,
		predeclared: starlark.StringDict{},
	}
	tr.thread = &starlark.Thread{
		Name:  name,
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	starlarktest.SetReporter(tr.thread, tr)

	tr.predeclared["eval"] = starlark.NewBuiltin("eval", tr.builtinEval)
	tr.predeclared["read_file"] = starlark.NewBuiltin("read_file", tr.builtinReadFile)
	tr.predeclared["txtar"] = starlark.NewBuiltin("txtar", tr.builtinTxtar)

	for k, v := range r.assertMod {
		tr.predeclared[k] = v
	}
	return tr
}

func (tr *testCaseRunner) Error(args ...any) {
	tr.failures = append(tr.failures, fmt.Sprint(args...))
}

// TestResult holds the outcome of a single Starlark test function.
type TestResult struct {
	Name     string
	Failures []string
}

// Passed returns true if the test had no failures.
func (tr *TestResult) Passed() bool {
	return len(tr.Failures) == 0
}

// Runner loads and executes Starlark test files.
type Runner struct {
	assertMod starlark.StringDict
}

// NewRunner creates a new test runner.
func NewRunner() *Runner {
	assertMod, err := starlarktest.LoadAssertModule()
	if err != nil {
		panic(fmt.Errorf("failed to load starlarktest assert module: %w", err))
	}
	return &Runner{
		assertMod: assertMod,
	}
}

// RunFile executes a single Starlark test file and returns the results
// for each test_* function found in it.
// src may be nil, or a string, []byte, or io.Reader containing the file source.
func (r *Runner) RunFile(filename string, src any) ([]TestResult, error) {
	baseDir := ""
	if filename != "" {
		baseDir = filepath.Dir(filename)
	}

	var fileSystem fs.FS
	if baseDir != "" {
		fileSystem = os.DirFS(baseDir)
	}

	tr := r.newTestCaseRunner("ruletest", fileSystem)

	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, tr.thread, filename, src, tr.predeclared)
	if err != nil {
		if evalErr, ok := errors.AsType[*starlark.EvalError](err); ok {
			return nil, fmt.Errorf("loading %s: %w\n%s", filename, err, evalErr.Backtrace())
		}
		return nil, fmt.Errorf("loading %s: %w", filename, err)
	}

	testFns := make(map[string]*starlark.Function)
	for _, name := range globals.Keys() {
		if !strings.HasPrefix(name, "test_") {
			continue
		}
		fn, ok := globals[name].(*starlark.Function)
		if !ok {
			return nil, fmt.Errorf("expected %s to be a function, got %s", name, globals[name].Type())
		}
		if fn.NumParams() != 0 {
			return nil, fmt.Errorf("expected %s to have no parameters, got %d", name, fn.NumParams())
		}
		testFns[name] = fn
	}

	var results []TestResult
	for name, fn := range testFns {
		result := r.runOneTest(name, fn, fileSystem)
		results = append(results, result)
	}

	return results, nil
}

func (r *Runner) runOneTest(name string, fn *starlark.Function, fileSystem fs.FS) TestResult {
	tr := r.newTestCaseRunner(name, fileSystem)
	result := TestResult{Name: name}

	_, err := starlark.Call(tr.thread, fn, nil, nil)
	if err != nil {
		if evalErr, ok := errors.AsType[*starlark.EvalError](err); ok {
			result.Failures = append(result.Failures, evalErr.Backtrace())
		} else {
			result.Failures = append(result.Failures, err.Error())
		}
	}

	result.Failures = append(result.Failures, tr.failures...)

	if strings.HasPrefix(name, "test_fail_") {
		if len(result.Failures) == 0 {
			result.Failures = []string{"expected test to fail, but it succeeded"}
		} else {
			result.Failures = nil // test failed as expected
		}
	}

	return result
}

// DiscoverFiles walks the given directory tree and returns the paths of
// all *.star files found, in sorted order.
func DiscoverFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".star") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", root, err)
	}
	sort.Strings(files)
	return files, nil
}

// RunDir discovers and executes all *.star test files under the given
// directory, reporting results through t.
func (r *Runner) RunDir(t *testing.T, dir string) {
	t.Helper()

	files, err := DiscoverFiles(dir)
	if err != nil {
		t.Fatalf("discovering test files: %v", err)
	}

	if len(files) == 0 {
		t.Logf("no *.star test files found in %s", dir)
		return
	}

	for _, file := range files {
		rel, err := filepath.Rel(dir, file)
		if err != nil {
			t.Fatalf("failed to compute relative path for %s: %v", file, err)
		}

		t.Run(rel, func(t *testing.T) {
			results, err := r.RunFile(file, nil)
			if err != nil {
				t.Fatalf("running %s: %v", file, err)
			}

			for _, result := range results {
				t.Run(result.Name, func(t *testing.T) {
					for _, msg := range result.Failures {
						t.Error(msg)
					}
				})
			}
		})
	}
}
