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

// TestDryRun_AllCasesSkipped verifies that DryRun exits cleanly when every
// test case in the fixture has a skip_reason.  This is the expected state for
// rules that depend on git commit history: you scaffold the fixture, mark all
// cases as skipped, and come back to fill them in later.  DryRun should not
// treat a fully-skipped fixture as an error.
func TestDryRun_AllCasesSkipped_NoErrors(t *testing.T) {
	t.Parallel()

	yaml := `
version: v1
rule_name: git-history-rule
test_cases:
  - name: "needs default branch name"
    skip_reason: "requires git branch metadata, not yet supported by memfs"
  - name: "needs commit count"
    skip_reason: "requires git commit history, not yet supported by memfs"
  - name: "needs tag information"
    skip_reason: "requires git tags, not yet supported by memfs"
`
	results, err := DryRun(writeTempFixture(t, yaml))
	if err != nil {
		t.Fatalf("DryRun returned an unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Skipped {
			t.Errorf("case %q should be skipped but was not", r.Name)
		}
		if r.SkipReason == "" {
			t.Errorf("case %q should have a SkipReason set", r.Name)
		}
		if r.Err != nil {
			t.Errorf("case %q should have no error, got: %v", r.Name, r.Err)
		}
	}
}

// TestDryRun_SkipReasonPreservedVerbatim confirms that the exact text written
// in skip_reason comes back in the Result.  The runner surfaces this text in
// CI output so that contributors know exactly what needs to be done to un-skip
// the case, so it must not be truncated or modified.
func TestDryRun_SkipReasonPreservedVerbatim(t *testing.T) {
	t.Parallel()

	wantReason := "requires git commit history, not yet supported by memfs"

	yaml := `
version: v1
rule_name: some-rule
test_cases:
  - name: "skipped case"
    skip_reason: "requires git commit history, not yet supported by memfs"
`
	results, err := DryRun(writeTempFixture(t, yaml))
	if err != nil {
		t.Fatalf("DryRun returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SkipReason != wantReason {
		t.Errorf("SkipReason = %q, want %q", results[0].SkipReason, wantReason)
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
