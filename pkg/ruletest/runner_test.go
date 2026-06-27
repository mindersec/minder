// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDiscoverFiles(t *testing.T) {
	t.Parallel()
	files, err := DiscoverFiles("testdata")
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	found := make(map[string]bool)
	for _, f := range files {
		found[f] = true
	}

	expected := []string{
		filepath.Join("testdata", "eval.star"),
		filepath.Join("testdata", "sample.star"),
	}

	for _, exp := range expected {
		if !found[exp] {
			t.Errorf("expected to find %s, but it was missing", exp)
		}
	}
}

func TestRunFile(t *testing.T) {
	t.Parallel()
	r := NewRunner()
	results, err := r.RunFile(filepath.Join("testdata", "sample.star"), nil, nil)
	if err != nil {
		t.Fatalf("RunFile failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 test functions to be discovered and run, got %d", len(results))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	tc := []struct {
		name       string
		wantPassed bool
		failures   []string
	}{
		{
			name:       "test_exception",
			wantPassed: false,
		},
		{
			name:       "test_failing",
			wantPassed: false,
			failures:   []string{"this test failed intentionally"},
		},
		{
			name:       "test_passing",
			wantPassed: true,
		},
	}

	for i, tt := range tc {
		if results[i].Name != tt.name {
			t.Errorf("results[%d]: expected name %s, got %s", i, tt.name, results[i].Name)
		}
		if results[i].Passed() != tt.wantPassed {
			t.Errorf("results[%d] (%s): expected passed=%v, got passed=%v", i, tt.name, tt.wantPassed, results[i].Passed())
		}
		if !tt.wantPassed && len(results[i].Failures) == 0 {
			t.Errorf("results[%d] (%s): expected failures but got none", i, tt.name)
		}
		for j, want := range tt.failures {
			if j >= len(results[i].Failures) {
				t.Errorf("results[%d] (%s): missing expected failure %q", i, tt.name, want)
				continue
			}
			if !strings.Contains(results[i].Failures[j], want) {
				t.Errorf("results[%d] (%s): failure[%d] = %q, want it to contain %q", i, tt.name, j, results[i].Failures[j], want)
			}
		}
	}
}

func TestRunEvalFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
	}{
		{name: "eval", file: "eval.star"},
		{name: "builtins", file: "builtins_test.star"},
	}

	r := NewRunner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results, err := r.RunFile(filepath.Join("testdata", tt.file), nil, nil)
			if err != nil {
				t.Fatalf("RunFile failed for %s: %v", tt.file, err)
			}
			for _, res := range results {
				if strings.HasPrefix(res.Name, "test_fail_") {
					if res.Passed() {
						t.Errorf("expected test %s to fail, but it passed", res.Name)
					}
					continue
				}
				if !res.Passed() {
					t.Errorf("test %s failed: %v", res.Name, res.Failures)
				}
			}
		})
	}
}

func TestTestDir(t *testing.T) {
	t.Parallel()
	r := NewRunner()
	dir := t.TempDir()
	content := []byte(`
def test_pass():
    assert.eq(1, 1)
`)
	if err := os.WriteFile(filepath.Join(dir, "pass.star"), content, 0644); err != nil {
		t.Fatal(err)
	}
	r.TestDir(t, dir)
}
