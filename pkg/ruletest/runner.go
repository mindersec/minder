// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ruletest provides a Starlark-based test runner for Minder rule types.
package ruletest

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarktest"
	"go.starlark.net/syntax"

	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
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
	Filename string
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
//
// filename is the path to the *.star file. If src is nil, the file is
// read from disk; otherwise src may be a string, []byte, or io.Reader
// containing the Starlark source. ruleTypes supplies the rule type
// definitions available to eval() calls within the test file.
func (r *Runner) RunFile(filename string, src any, ruleTypes map[string]*minderv1.RuleType) ([]TestResult, error) {
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	baseDir := filepath.Dir(filename)
	fileSystem := os.DirFS(baseDir)

	name := filepath.Base(filename)
	tr := r.newTestCaseRunner(name, fileSystem, ruleTypes)

	globals, err := tr.runFile(filename, src)
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

	base := filepath.Base(filename)
	var results []TestResult
	for name, fn := range testFns {
		result := r.runOneTest(name, fn, fileSystem, ruleTypes)
		result.Filename = base
		results = append(results, result)
	}

	return results, nil
}

func (r *Runner) runOneTest(
	name string,
	fn *starlark.Function,
	fileSystem fs.FS,
	ruleTypes map[string]*minderv1.RuleType,
) TestResult {
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

// loadRulesFromDir finds and parses all *.yaml files in the given directory
// into a map of RuleTypes keyed by rule name.
func loadRulesFromDir(dir string) (map[string]*minderv1.RuleType, error) {
	ruleTypes := make(map[string]*minderv1.RuleType)
	yamlFiles, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("globbing yaml files: %w", err)
	}
	for _, yf := range yamlFiles {
		rt, err := loadSingleRule(yf)
		if err != nil {
			continue // skip files that aren't valid rule types
		}
		if rt != nil && rt.Name != "" {
			if _, exists := ruleTypes[rt.Name]; exists {
				return nil, fmt.Errorf("duplicate rule type name %q in directory %s", rt.Name, dir)
			}
			ruleTypes[rt.Name] = rt
		}
	}
	return ruleTypes, nil
}

// loadSingleRule reads a single YAML file and returns the parsed RuleType, if any.
func loadSingleRule(path string) (*minderv1.RuleType, error) {
	decoder, closer := fileconvert.DecoderForFile(path)
	if decoder == nil {
		return nil, fmt.Errorf("error opening file: %s", path)
	}
	defer func(c io.Closer) {
		_ = c.Close()
	}(closer)
	return fileconvert.ReadResourceTyped[*minderv1.RuleType](decoder)
}

// RunPaths takes a list of file or directory paths, discovering all *.star test
// files recursively. Tests are grouped by their immediate directory, and any
// *.yaml rules in that same directory are loaded and made available to the tests.
// It collects errors instead of returning early on the first error.
func (r *Runner) RunPaths(paths []string) ([]TestResult, error) {
	expanded, err := util.ExpandFileArgs(paths...)
	if err != nil {
		return nil, fmt.Errorf("expanding paths: %w", err)
	}

	// Group .star files by their immediate directory
	filesByDir := make(map[string][]string)
	for _, f := range expanded {
		if !f.Expanded && filepath.Ext(f.Path) != ".star" && filepath.Ext(f.Path) != ".yaml" && filepath.Ext(f.Path) != ".yml" {
			// If it's a specific file that is not a star or yaml file, we skip it
			continue
		}
		if filepath.Ext(f.Path) == ".star" {
			dir := filepath.Dir(f.Path)
			filesByDir[dir] = append(filesByDir[dir], f.Path)
		}
	}

	var allResults []TestResult
	var errs []error

	// Ensure deterministic execution order by sorting directories
	dirs := slices.Sorted(maps.Keys(filesByDir))

	for _, dir := range dirs {
		files := filesByDir[dir]
		sort.Strings(files)

		ruleTypes, err := loadRulesFromDir(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("loading rules for directory %s: %w", dir, err))
			continue
		}

		for _, file := range files {
			res, err := r.RunFile(file, nil, ruleTypes)
			if err != nil {
				errs = append(errs, fmt.Errorf("error running file %s: %w", file, err))
				continue
			}
			allResults = append(allResults, res...)
		}
	}
	return allResults, errors.Join(errs...)
}
