// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"gopkg.in/yaml.v3"
)

// --- Test-local types and helpers (self-contained) ---

type fixture struct {
	Version   string     `yaml:"version"`
	RuleName  string     `yaml:"rule_name"`
	TestCases []TestCase `yaml:"test_cases"`
}

// TestCase describes one scenario inside a fixture file.
type TestCase struct {
	Name       string             `yaml:"name"`
	Expect     string             `yaml:"expect"`
	SkipReason string             `yaml:"skip_reason"`
	MockData   ProviderMockConfig `yaml:"mock_data"`
}

// ProviderMockConfig holds mock data for all provider types.
type ProviderMockConfig struct {
	GitFiles            map[string]string           `yaml:"git_files"`
	HTTPResponses       map[string]HTTPResponseMock `yaml:"http_responses"`
	DataSourceResponses map[string]HTTPResponseMock `yaml:"data_source_responses"`
}

// HTTPResponseMock defines a canned HTTP response for a given URL.
type HTTPResponseMock struct {
	StatusCode int    `yaml:"status_code"`
	Body       string `yaml:"body"`
}

// Mocks holds the constructed mock objects for a single test case.
type Mocks struct {
	GitFilesystem    billy.Filesystem
	HTTPClient       *http.Client
	DataSourceClient *http.Client
}

// Result records the outcome of a single test case after DryRun.
type Result struct {
	Name       string
	Err        error
	Skipped    bool
	SkipReason string
}

type mockRoundTripper struct {
	responses map[string]HTTPResponseMock
}

// NewMockRoundTripper creates a mockRoundTripper from the given map.
func NewMockRoundTripper(responses map[string]HTTPResponseMock) *mockRoundTripper {
	return &mockRoundTripper{responses: responses}
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.URL.String()
	if mock, ok := m.responses[key]; ok {
		return &http.Response{
			StatusCode: mock.StatusCode,
			Body:       io.NopCloser(strings.NewReader(mock.Body)),
			Header:     make(http.Header),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("no mock for URL: %s", key))),
		Header:     make(http.Header),
	}, nil
}

func parsePath(path string) (*fixture, error) {
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

func newMockBillyFS(files map[string]string) (billy.Filesystem, error) {
	fs := memfs.New()
	for path, content := range files {
		f, err := fs.Create(path)
		if err != nil {
			return nil, fmt.Errorf("creating %q: %w", path, err)
		}
		if _, writeErr := f.Write([]byte(content)); writeErr != nil {
			f.Close()
			return nil, fmt.Errorf("writing %q: %w", path, writeErr)
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("closing %q: %w", path, err)
		}
	}
	return fs, nil
}

// BuildMocks constructs the full set of mock objects for a test case.
func BuildMocks(tc TestCase) (*Mocks, error) {
	m := &Mocks{}
	if len(tc.MockData.GitFiles) > 0 {
		fs, err := newMockBillyFS(tc.MockData.GitFiles)
		if err != nil {
			return nil, fmt.Errorf("case %q: building git filesystem: %w", tc.Name, err)
		}
		m.GitFilesystem = fs
	} else {
		m.GitFilesystem = memfs.New()
	}
	if len(tc.MockData.HTTPResponses) > 0 {
		m.HTTPClient = &http.Client{Transport: NewMockRoundTripper(tc.MockData.HTTPResponses)}
	} else {
		m.HTTPClient = &http.Client{Transport: NewMockRoundTripper(nil)}
	}
	if len(tc.MockData.DataSourceResponses) > 0 {
		m.DataSourceClient = &http.Client{Transport: NewMockRoundTripper(tc.MockData.DataSourceResponses)}
	} else {
		m.DataSourceClient = &http.Client{Transport: NewMockRoundTripper(nil)}
	}
	return m, nil
}

// DryRun parses a fixture file and validates each test case.
func DryRun(path string) ([]Result, error) {
	fx, err := parsePath(path)
	if err != nil {
		return nil, fmt.Errorf("dry-run: %w", err)
	}
	results := make([]Result, 0, len(fx.TestCases))
	for _, tc := range fx.TestCases {
		r := Result{Name: tc.Name}
		if tc.SkipReason != "" {
			r.Skipped = true
			r.SkipReason = tc.SkipReason
			results = append(results, r)
			continue
		}
		_, buildErr := BuildMocks(tc)
		if buildErr != nil {
			r.Err = buildErr
			results = append(results, r)
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

// TestBuildMocks_AllThreeMockTypes exercises a fixture test case that uses
// all three ingestion paths at once: git_files, http_responses, and
// data_source_responses.  This is the realistic scenario for a complex rule
// that reads a file from the repository AND calls a REST API AND reads from a
// declared data source.
func TestBuildMocks_AllThreeMockTypes(t *testing.T) {
	t.Parallel()

	tc := TestCase{
		Name:   "all three mock types",
		Expect: "pass",
		MockData: ProviderMockConfig{
			GitFiles: map[string]string{
				"SECURITY.md": "Please report vulnerabilities to security@example.com",
			},
			HTTPResponses: map[string]HTTPResponseMock{
				"https://api.github.com/repos/owner/repo/vulnerability-alerts": {
					StatusCode: 200,
					Body:       `{"enabled": true}`,
				},
			},
			DataSourceResponses: map[string]HTTPResponseMock{
				"https://ds.example.com/org-policy": {
					StatusCode: 200,
					Body:       `{"require_security_md": true}`,
				},
			},
		},
	}

	mocks, err := BuildMocks(tc)
	if err != nil {
		t.Fatalf("BuildMocks returned unexpected error: %v", err)
	}

	// Git filesystem contains the declared file.
	f, err := mocks.GitFilesystem.Open("SECURITY.md")
	if err != nil {
		t.Fatalf("opening SECURITY.md: %v", err)
	}
	content, _ := io.ReadAll(f)
	f.Close()
	if string(content) != "Please report vulnerabilities to security@example.com" {
		t.Errorf("git file content = %q", string(content))
	}

	// HTTP client serves the vulnerability alerts response.
	req, _ := http.NewRequest(http.MethodGet,
		"https://api.github.com/repos/owner/repo/vulnerability-alerts", nil)
	resp, err := mocks.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP client request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("http status = %d, want 200", resp.StatusCode)
	}

	// Data source client serves the org-policy response.
	req2, _ := http.NewRequest(http.MethodGet, "https://ds.example.com/org-policy", nil)
	resp2, err := mocks.DataSourceClient.Do(req2)
	if err != nil {
		t.Fatalf("DataSource client request: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Errorf("data source status = %d, want 200", resp2.StatusCode)
	}

	// Data source client does NOT accidentally serve HTTP provider URLs.
	req3, _ := http.NewRequest(http.MethodGet,
		"https://api.github.com/repos/owner/repo/vulnerability-alerts", nil)
	resp3, err := mocks.DataSourceClient.Do(req3)
	if err != nil {
		t.Fatalf("DataSource client unexpected error: %v", err)
	}
	if resp3.StatusCode == 200 {
		t.Error("data source client should not serve HTTP provider URLs")
	}
}

// TestDryRun_AllThreeMockTypes runs DryRun on a fixture that uses all three
// ingestion types, ensuring the full validation pipeline handles them.
func TestDryRun_AllThreeMockTypes_PassesValidation(t *testing.T) {
	t.Parallel()

	yaml := `
version: v1
rule_name: complex-rule
test_cases:
  - name: "all three mocks pass"
    expect: pass
    mock_data:
      git_files:
        "SECURITY.md": "security policy content"
      http_responses:
        "https://api.github.com/repos/owner/repo/vulnerability-alerts":
          status_code: 200
          body: '{"enabled": true}'
      data_source_responses:
        "https://ds.example.com/org-policy":
          status_code: 200
          body: '{"require_security_md": true}'
`
	results, err := DryRun(writeTempFixture(t, yaml))
	if err != nil {
		t.Fatalf("DryRun returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("expected no error, got: %v", results[0].Err)
	}
	if results[0].Skipped {
		t.Error("case should not be skipped")
	}
}

func writeTempFixture(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "fixture-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}
