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

type Fixture struct {
	Version   string     `yaml:"version"`
	RuleName  string     `yaml:"rule_name"`
	TestCases []TestCase `yaml:"test_cases"`
}

type TestCase struct {
	Name       string             `yaml:"name"`
	Expect     string             `yaml:"expect"`
	SkipReason string             `yaml:"skip_reason"`
	MockData   ProviderMockConfig `yaml:"mock_data"`
}

type ProviderMockConfig struct {
	GitFiles            map[string]string           `yaml:"git_files"`
	HTTPResponses       map[string]HTTPResponseMock `yaml:"http_responses"`
	DataSourceResponses map[string]HTTPResponseMock `yaml:"data_source_responses"`
}

type HTTPResponseMock struct {
	StatusCode int    `yaml:"status_code"`
	Body       string `yaml:"body"`
}

type Mocks struct {
	GitFilesystem    billy.Filesystem
	HTTPClient       *http.Client
	DataSourceClient *http.Client
}

type Result struct {
	Name       string
	Err        error
	Skipped    bool
	SkipReason string
}

type MockRoundTripper struct {
	responses map[string]HTTPResponseMock
}

func NewMockRoundTripper(responses map[string]HTTPResponseMock) *MockRoundTripper {
	return &MockRoundTripper{responses: responses}
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
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

func Parse(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading fixture %s: %w", path, err)
	}

	var f Fixture
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing fixture %s: %w", path, err)
	}

	return &f, nil
}

func NewMockBillyFS(files map[string]string) (billy.Filesystem, error) {
	fs := memfs.New()
	for path, content := range files {
		f, err := fs.Create(path)
		if err != nil {
			return nil, fmt.Errorf("creating %q: %w", path, err)
		}
		if _, writeErr := f.Write([]byte(content)); writeErr != nil {
			if closeErr := f.Close(); closeErr != nil {
				return nil, fmt.Errorf("writing %q: %w (also failed to close: %v)", path, writeErr, closeErr)
			}
			return nil, fmt.Errorf("writing %q: %w", path, writeErr)
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("closing %q: %w", path, err)
		}
	}
	return fs, nil
}

func BuildMocks(tc TestCase) (*Mocks, error) {
	m := &Mocks{}

	if len(tc.MockData.GitFiles) > 0 {
		fs, err := NewMockBillyFS(tc.MockData.GitFiles)
		if err != nil {
			return nil, fmt.Errorf("case %q: building git filesystem: %w", tc.Name, err)
		}
		m.GitFilesystem = fs
	} else {
		m.GitFilesystem = memfs.New()
	}

	if len(tc.MockData.HTTPResponses) > 0 {
		m.HTTPClient = &http.Client{
			Transport: NewMockRoundTripper(tc.MockData.HTTPResponses),
		}
	} else {
		m.HTTPClient = &http.Client{
			Transport: NewMockRoundTripper(nil),
		}
	}

	if len(tc.MockData.DataSourceResponses) > 0 {
		m.DataSourceClient = &http.Client{
			Transport: NewMockRoundTripper(tc.MockData.DataSourceResponses),
		}
	} else {
		m.DataSourceClient = &http.Client{
			Transport: NewMockRoundTripper(nil),
		}
	}

	return m, nil
}

func DryRun(path string) ([]Result, error) {
	fixture, err := Parse(path)
	if err != nil {
		return nil, fmt.Errorf("dry-run: %w", err)
	}

	results := make([]Result, 0, len(fixture.TestCases))
	for _, tc := range fixture.TestCases {
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

// TestDryRun_ResultCountMatchesTestCaseCount is a contract test.
// DryRun must return exactly one Result per test case in the fixture,
// regardless of whether the case passed, failed, was skipped, or hit an error.
// CI tooling that counts pass/fail/skip ratios depends on this guarantee.
func TestDryRun_ResultCountMatchesTestCaseCount(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		caseCount int
		yaml      string
	}{
		{
			name:      "one case",
			caseCount: 1,
			yaml: `
version: v1
rule_name: some-rule
test_cases:
  - name: "only case"
    expect: pass
    mock_data:
      git_files:
        "SECURITY.md": "content"
`,
		},
		{
			name:      "three cases mixed",
			caseCount: 3,
			yaml: `
version: v1
rule_name: some-rule
test_cases:
  - name: "passes"
    expect: pass
    mock_data:
      git_files:
        "SECURITY.md": "content"
  - name: "skipped"
    skip_reason: "not yet supported"
  - name: "fails"
    expect: fail
    mock_data:
      git_files:
        "README.md": "hello"
`,
		},
		{
			name:      "five cases all skipped",
			caseCount: 5,
			yaml: `
version: v1
rule_name: some-rule
test_cases:
  - name: "skipped 1"
    skip_reason: "reason"
  - name: "skipped 2"
    skip_reason: "reason"
  - name: "skipped 3"
    skip_reason: "reason"
  - name: "skipped 4"
    skip_reason: "reason"
  - name: "skipped 5"
    skip_reason: "reason"
`,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results, err := DryRun(writeTempFixture(t, tt.yaml))
			if err != nil {
				t.Fatalf("DryRun returned error: %v", err)
			}
			if len(results) != tt.caseCount {
				t.Errorf("len(results) = %d, want %d", len(results), tt.caseCount)
			}
		})
	}
}

// TestDryRun_ResultNameMatchesCaseName confirms that the Name field in each
// Result matches the test case name from the fixture in the same position.
// CI output shows this name, so it must round-trip cleanly.
func TestDryRun_ResultNameMatchesCaseName(t *testing.T) {
	t.Parallel()

	yaml := `
version: v1
rule_name: some-rule
test_cases:
  - name: "first case"
    expect: pass
    mock_data:
      git_files:
        "SECURITY.md": "content"
  - name: "second case"
    skip_reason: "not yet supported"
`
	results, err := DryRun(writeTempFixture(t, yaml))
	if err != nil {
		t.Fatalf("DryRun returned error: %v", err)
	}

	wantNames := []string{"first case", "second case"}
	for i, r := range results {
		if r.Name != wantNames[i] {
			t.Errorf("results[%d].Name = %q, want %q", i, r.Name, wantNames[i])
		}
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
