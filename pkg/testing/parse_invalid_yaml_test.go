// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// --- Test-local types and helpers ---

type fixture struct {
	Version   string     `yaml:"version"`
	RuleName  string     `yaml:"rule_name"`
	TestCases []testCase `yaml:"test_cases"`
}

type testCase struct {
	Name       string             `yaml:"name"`
	Expect     string             `yaml:"expect"`
	SkipReason string             `yaml:"skip_reason"`
	MockData   providerMockConfig `yaml:"mock_data"`
}

type providerMockConfig struct {
	GitFiles            map[string]string           `yaml:"git_files"`
	HTTPResponses       map[string]httpResponseMock `yaml:"http_responses"`
	DataSourceResponses map[string]httpResponseMock `yaml:"data_source_responses"`
}

type httpResponseMock struct {
	StatusCode int    `yaml:"status_code"`
	Body       string `yaml:"body"`
}

// Parse reads a fixture YAML file from disk and returns the parsed fixture.
func Parse(path string) (*fixture, error) {
	//nolint:gosec // path is provided by test fixtures, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading fixture %s: %w", path, err)
	}
	var f fixture
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing fixture %s: %w", path, err)
	}
	return &f, nil
}

// TestParse_MalformedYAML covers the yaml.Unmarshal error branch inside Parse.
// The YAML parser rejects the triple-brace sequence because it is not valid
// YAML flow-mapping syntax, so Unmarshal returns an error before validation
// even runs.
func TestParse_MalformedYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")

	// Write bytes that look like a YAML file but are syntactically broken.
	// The unclosed flow mapping "{{{" triggers a parse error in yaml.v3.
	if err := os.WriteFile(path, []byte("{{{this is not valid yaml"), 0o644); err != nil {
		t.Fatalf("writing bad fixture file: %v", err)
	}

	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected an error for malformed YAML, got nil")
	}
}

// TestParse_BinaryContent covers the same branch with non-text bytes.
// Rule fixture files should always be plain text; passing binary data is
// another way the YAML parser can fail.
func TestParse_BinaryContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "binary.yaml")

	// Write raw bytes that cannot form valid YAML.
	if err := os.WriteFile(path, []byte{0x80, 0x81, 0x82, 0xff, 0xfe}, 0o644); err != nil {
		t.Fatalf("writing binary fixture file: %v", err)
	}

	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected an error for binary content, got nil")
	}
}
