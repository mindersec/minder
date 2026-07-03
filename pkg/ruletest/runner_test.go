// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"strings"
	"testing"
)

func TestTestDir(t *testing.T) {
	t.Parallel()
	r := NewRunner()
	testDir(t, r, "testdata")
}

// testDir discovers and executes all *.star test files under the given
// directory, reporting results through t. It also loads *.yaml rules.
func testDir(t *testing.T, r *Runner, dir string) {
	t.Helper()

	results, err := r.RunPaths([]string{dir})
	if err != nil {
		t.Fatalf("running tests in %s: %v", dir, err)
	}

	if len(results) == 0 {
		t.Logf("no *.star test files found in %s", dir)
		return
	}

	for _, result := range results {
		result := result
		name := result.Filename + "/" + result.Name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if strings.HasPrefix(result.Name, "test_fail_") {
				if result.Passed() {
					t.Errorf("expected test %s to fail, but it passed", result.Name)
				}
				return
			}
			for _, msg := range result.Failures {
				t.Error(msg)
			}
		})
	}
}
