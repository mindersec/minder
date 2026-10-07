// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	for _, run := range results {
		for _, result := range run.Results {
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
				for _, msg := range result.Errors {
					t.Error(msg)
				}
			})
		}
	}
}

func TestLoadRulesFromDir(t *testing.T) {
	t.Parallel()

	sample, err := os.ReadFile("testdata/rule_type_sample.yaml")
	require.NoError(t, err)
	dataSource, err := os.ReadFile("testdata/mock_datasource_def.yaml")
	require.NoError(t, err)
	broken := strings.Replace(string(sample), "name: branch_protection_reviews", "name: broken_rule", 1) +
		"remediate:\n  type: rest\n"

	dir := t.TempDir()
	files := map[string]string{
		"valid_rule.yaml":  string(sample),
		"data_source.yaml": string(dataSource),
		"compose.yaml":     "type: service\nimage: nginx\n",
		"no_type.yaml":     "name: something\nversion: v1\n",
		"empty.yaml":       "",
		"broken_rule.yaml": broken,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}

	ruleTypes, err := loadRulesFromDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken_rule.yaml")
	assert.Contains(t, err.Error(), `unknown field "remediate"`)
	for _, skipped := range []string{"data_source.yaml", "compose.yaml", "no_type.yaml", "empty.yaml"} {
		assert.NotContains(t, err.Error(), skipped)
	}
	assert.Contains(t, ruleTypes, "branch_protection_reviews")
	assert.NotContains(t, ruleTypes, "broken_rule")
}

func TestLoadRulesFromDirEmptyRego(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "empty.rego"), nil, 0o600))

	_, err := loadRulesFromDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty.rego")
}
