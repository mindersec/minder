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

// TestDryRun_BuildMocksFailure covers the branch inside DryRun where
// BuildMocks returns an error.  DryRun should not stop processing; it should
// record the failing case in the results and continue to the next one.
// This matters for CI: if one fixture entry is broken you still want to see
// all the other failures, not just the first.
func TestDryRun_BuildMocksFailure_RecordedInResults(t *testing.T) {
	t.Parallel()

	// An empty-string git_files key makes BuildMocks fail because the
	// in-memory filesystem refuses to create a file at that path.
	yaml := `
version: v1
rule_name: some-rule
test_cases:
  - name: "broken case"
    expect: pass
    mock_data:
      git_files:
        "": "this path is invalid"
`
	results, err := DryRun(writeTempFixture(t, yaml))

	// DryRun itself should not return a top-level error; the fixture is valid.
	if err != nil {
		t.Fatalf("DryRun returned unexpected top-level error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Error("expected the result to carry an error for the broken case, got nil")
	}
	if results[0].Skipped {
		t.Error("a case with a build error should not be marked as skipped")
	}
}

// TestDryRun_MixedBrokenAndHealthyCases confirms that DryRun keeps going
// after a BuildMocks failure and records results for all subsequent cases.
func TestDryRun_MixedBrokenAndHealthyCases(t *testing.T) {
	t.Parallel()

	yaml := `
version: v1
rule_name: some-rule
test_cases:
  - name: "broken case"
    expect: pass
    mock_data:
      git_files:
        "": "bad path"
  - name: "healthy case"
    expect: pass
    mock_data:
      git_files:
        "SECURITY.md": "report vulns here"
`
	results, err := DryRun(writeTempFixture(t, yaml))
	if err != nil {
		t.Fatalf("DryRun returned unexpected top-level error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Error("first case should have an error")
	}
	if results[1].Err != nil {
		t.Errorf("second case should have no error, got: %v", results[1].Err)
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
