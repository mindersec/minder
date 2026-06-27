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

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
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
	ruleTypes   map[string]*minderv1.RuleType
}

func (r *Runner) newTestCaseRunner(name string, fileSystem fs.FS, ruleTypes map[string]*minderv1.RuleType) *testCaseRunner {
	if fileSystem == nil {
		panic("fileSystem cannot be nil")
	}
	tr := &testCaseRunner{
		fs:          fileSystem,
		predeclared: starlark.StringDict{},
		ruleTypes:   ruleTypes,
	}
	tr.thread = &starlark.Thread{
		Name:  name,
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}
	starlarktest.SetReporter(tr.thread, tr)

	tr.predeclared["eval"] = starlark.NewBuiltin("eval", tr.builtinEval)
	tr.predeclared["read_file"] = starlark.NewBuiltin("read_file", tr.builtinReadFile)
	tr.predeclared["txtar"] = starlark.NewBuiltin("txtar", builtinTxtar)
	tr.predeclared["body"] = starlark.NewBuiltin("body", builtinBody)
	tr.predeclared["code"] = starlark.NewBuiltin("code", builtinCode)

	for k, v := range r.assertMod {
		tr.predeclared[k] = v
	}
	return tr
}

// runFile loads and executes a Starlark file within the context of the testCaseRunner.
func (tr *testCaseRunner) runFile(filename string, src any) (starlark.StringDict, error) {
	return starlark.ExecFileOptions(&syntax.FileOptions{}, tr.thread, filename, src, tr.predeclared)
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

// RunFile executes a single Starlark test file. If src is non-nil, it is
// used as the file contents.
func (r *Runner) RunFile(filename string, src any) ([]TestResult, error) {
	return r.runFileWithRules(filename, src, nil)
}

func (r *Runner) runFileWithRules(filename string, src any, ruleTypes map[string]*minderv1.RuleType) ([]TestResult, error) {
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	baseDir := filepath.Dir(filename)
	fileSystem := os.DirFS(baseDir)

	name := filepath.Base(filename)
	tr := r.newTestCaseRunner(name, fileSystem, ruleTypes)

	globals, err := tr.runFile(filename, src)
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
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
		result := r.runOneTest(name, fn, fileSystem, ruleTypes)
		results = append(results, result)
	}

	return results, nil
}

func (r *Runner) runOneTest(name string, fn *starlark.Function, fileSystem fs.FS, ruleTypes map[string]*minderv1.RuleType) TestResult {
	tr := r.newTestCaseRunner(name, fileSystem, ruleTypes)
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

// loadRulesFromDir finds and parses all *.yaml files in the given directory
// into a map of RuleTypes keyed by rule name.
func loadRulesFromDir(dir string) (map[string]*minderv1.RuleType, error) {
	ruleTypes := make(map[string]*minderv1.RuleType)
	yamlFiles, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("globbing yaml files: %w", err)
	}
	for _, yf := range yamlFiles {
		decoder, closer := fileconvert.DecoderForFile(yf)
		if decoder == nil {
			return nil, fmt.Errorf("error opening file: %s", yf)
		}
		rt, err := fileconvert.ReadResourceTyped[*minderv1.RuleType](decoder)
		closer.Close()
		if err == nil && rt != nil && rt.Name != "" {
			ruleTypes[rt.Name] = rt
		}
	}
	return ruleTypes, nil
}

// RunDir discovers and executes all *.star test files under the given
// directory. It also discovers and loads any *.yaml rule files in the directory.
func (r *Runner) RunDir(dir string) ([]TestResult, error) {
	ruleTypes, err := loadRulesFromDir(dir)
	if err != nil {
		return nil, fmt.Errorf("loading rules: %w", err)
	}

	files, err := DiscoverFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("discovering test files: %w", err)
	}

	var allResults []TestResult
	for _, file := range files {
		results, err := r.runFileWithRules(file, nil, ruleTypes)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// TestDir discovers and executes all *.star test files under the given
// directory, reporting results through t. It also loads *.yaml rules.
func (r *Runner) TestDir(t *testing.T, dir string) {
	t.Helper()

	ruleTypes, err := loadRulesFromDir(dir)
	if err != nil {
		t.Fatalf("loading rules: %v", err)
	}

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
			results, err := r.runFileWithRules(file, nil, ruleTypes)
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
