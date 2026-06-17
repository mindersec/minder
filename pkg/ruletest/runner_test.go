// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"path/filepath"
	"sort"
	"testing"
)

func TestDiscoverFiles(t *testing.T) {
	t.Parallel()
	files, err := DiscoverFiles("testdata")
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	expected := filepath.Join("testdata", "sample.star")
	if files[0] != expected {
		t.Errorf("expected %s, got %s", expected, files[0])
	}
}

func TestRunFile(t *testing.T) {
	t.Parallel()
	r := NewRunner()
	results, err := r.RunFile(filepath.Join("testdata", "sample.star"), nil)
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
			if results[i].Failures[j] != want {
				t.Errorf("results[%d] (%s): failure[%d] = %q, want %q", i, tt.name, j, results[i].Failures[j], want)
			}
		}
	}
}
