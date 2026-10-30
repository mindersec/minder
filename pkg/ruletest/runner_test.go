// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/remote"
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

func TestPrintOutputCaptured(t *testing.T) {
	t.Parallel()
	r := NewRunner()
	results, err := r.RunFile("testdata/builtins_test.star", nil, nil)
	require.NoError(t, err)

	var printTest *TestResult
	for i := range results {
		if results[i].Name == "test_print_output_captured" {
			printTest = &results[i]
			break
		}
	}
	require.NotNil(t, printTest, "test_print_output_captured not found")
	assert.True(t, printTest.Passed())
	assert.Equal(t, []string{"hello from starlark", "second line"}, printTest.Output)
}

func TestPrintOutputIsolatedBetweenTests(t *testing.T) {
	t.Parallel()
	r := NewRunner()
	results, err := r.RunFile("testdata/builtins_test.star", nil, nil)
	require.NoError(t, err)

	outputs := make(map[string][]string)
	for _, res := range results {
		outputs[res.Name] = res.Output
	}

	assert.Equal(t, []string{"hello from starlark", "second line"}, outputs["test_print_output_captured"])
	assert.Equal(t, []string{"only in this test"}, outputs["test_print_other"])
}

func TestRegoPrintOutputCaptured(t *testing.T) {
	t.Parallel()
	r := NewRunner()

	ruleTypes, err := loadRulesFromDir("testdata")
	require.NoError(t, err)

	results, err := r.RunFile("testdata/print_test.star", nil, ruleTypes)
	require.NoError(t, err)

	var regoTest *TestResult
	for i := range results {
		if results[i].Name == "test_rego_print_captured" {
			regoTest = &results[i]
			break
		}
	}
	require.NotNil(t, regoTest, "test_rego_print_captured not found")
	assert.Contains(t, regoTest.Output, "hello from rego")
}

func TestEvaluatedRulesTrackedCorrectly(t *testing.T) {
	t.Parallel()
	r := NewRunner()

	ruleTypes, err := loadRulesFromDir("testdata")
	require.NoError(t, err)

	results, err := r.RunFile("testdata/print_test.star", nil, ruleTypes)
	require.NoError(t, err)

	var regoTest *TestResult
	for i := range results {
		if results[i].Name == "test_rego_print_captured" {
			regoTest = &results[i]
			break
		}
	}
	require.NotNil(t, regoTest, "test_rego_print_captured not found")
	assert.Contains(t, regoTest.EvaluatedRules, "print_rule",
		"EvaluatedRules should contain print_rule after eval() call")

	run := TestRun{
		LoadedRules: []string{"print_rule"},
		Results:     results,
	}
	assert.Empty(t, run.UncoveredRules(), "print_rule should not appear as uncovered")
}

// noNetworkDir holds rule tests that must run with network access disabled.
const noNetworkDir = "testdata_nonetwork"

// failingTransport fails the test on any outbound HTTP request.
type failingTransport struct{ t *testing.T }

func (f failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.t.Errorf("unexpected network call to %s", req.URL)
	return nil, fmt.Errorf("network access disabled in test: %s", req.URL)
}

// TestMultiArchNoNetwork runs the artifact end-to-end tests with outbound
// HTTP disabled, to prove that verification comes from the TestKit
// fixtures and not from a real sigstore TUF root or registry.
//
// The files live in testdata_nonetwork rather than testdata, so they only
// run here under the guard and not in TestTestDir's unguarded sweep.
//
// Not parallel: it swaps http.DefaultTransport, which the sigstore TUF
// client uses through http.DefaultClient, and remote.DefaultTransport,
// which go-containerregistry uses for registry calls. Parallel top-level
// tests stay paused until this test returns.
//
//nolint:paralleltest // swaps process-wide globals
func TestMultiArchNoNetwork(t *testing.T) {
	origHTTP, origRemote := http.DefaultTransport, remote.DefaultTransport
	http.DefaultTransport = failingTransport{t: t}
	remote.DefaultTransport = failingTransport{t: t}
	t.Cleanup(func() {
		http.DefaultTransport = origHTTP
		remote.DefaultTransport = origRemote
	})

	ruleTypes, err := loadRulesFromDir(noNetworkDir)
	require.NoError(t, err)

	results, err := NewRunner().RunFile(filepath.Join(noNetworkDir, "multiarch_test.star"), nil, ruleTypes)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, res := range results {
		assert.True(t, res.Passed(), "%s: failures=%v errors=%v", res.Name, res.Failures, res.Errors)
	}
}
