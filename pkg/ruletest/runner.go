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

const threadContextKey = "ruletest.context"

type threadContext struct {
	fs       fs.FS
	failures []string
}

func getThreadContext(thread *starlark.Thread) *threadContext {
	ctx, ok := thread.Local(threadContextKey).(*threadContext)
	if !ok {
		ctx = &threadContext{}
		thread.SetLocal(threadContextKey, ctx)
	}
	return ctx
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
	predeclared starlark.StringDict
}

// NewRunner creates a new test runner with the default set of predeclared builtins.
func NewRunner() *Runner {
	assertMod, err := starlarktest.LoadAssertModule()
	if err != nil {
		panic(fmt.Errorf("failed to load starlarktest assert module: %w", err))
	}

	predeclared := starlark.StringDict{
		"eval":      starlark.NewBuiltin("eval", builtinEval),
		"read_file": starlark.NewBuiltin("read_file", builtinReadFile),
		"txtar":     starlark.NewBuiltin("txtar", builtinTxtar),
	}
	for k, v := range assertMod {
		predeclared[k] = v
	}

	return &Runner{
		predeclared: predeclared,
	}
}

// RunFile executes a single Starlark test file and returns the results
// for each test_* function found in it.
// src may be nil, or a string, []byte, or io.Reader containing the file source.
func (r *Runner) RunFile(filename string, src any) ([]TestResult, error) {
	baseDir := ""
	if filename != "" {
		baseDir = filepath.Dir(filename)
		if !filepath.IsAbs(baseDir) {
			if abs, err := filepath.Abs(baseDir); err == nil {
				baseDir = abs
			}
		}
	}

	var fileSystem fs.FS
	if baseDir != "" {
		fileSystem = os.DirFS(baseDir)
	}

	thread := r.newThread("ruletest", fileSystem)

	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, filename, src, r.predeclared)
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

	ctx := getThreadContext(thread)
	var results []TestResult
	for name, fn := range testFns {
		result := r.runOneTest(name, fn, ctx.fs)
		results = append(results, result)
	}

	return results, nil
}

func (*Runner) newThread(name string, fileSystem fs.FS) *starlark.Thread {
	thread := &starlark.Thread{
		Name:  name,
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	starlarktest.SetReporter(thread, threadReporter{thread})
	ctx := &threadContext{fs: fileSystem}
	thread.SetLocal(threadContextKey, ctx)
	return thread
}

func (r *Runner) runOneTest(name string, fn *starlark.Function, fileSystem fs.FS) TestResult {
	thread := r.newThread(name, fileSystem)

	result := TestResult{Name: name}

	_, err := starlark.Call(thread, fn, nil, nil)
	if err != nil {
		if evalErr, ok := errors.AsType[*starlark.EvalError](err); ok {
			result.Failures = append(result.Failures, evalErr.Backtrace())
		} else {
			result.Failures = append(result.Failures, err.Error())
		}
	}

	result.Failures = append(result.Failures, getFailures(thread)...)

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

func appendFailure(thread *starlark.Thread, msg string) {
	ctx := getThreadContext(thread)
	ctx.failures = append(ctx.failures, msg)
}

type threadReporter struct {
	thread *starlark.Thread
}

func (r threadReporter) Error(args ...any) {
	appendFailure(r.thread, fmt.Sprint(args...))
}

// getFailures retrieves the accumulated failure messages from the thread-local
// storage.
func getFailures(thread *starlark.Thread) []string {
	ctx := getThreadContext(thread)
	return ctx.failures
}
