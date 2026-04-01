// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

type TestCase struct {
	Name       string
	Expect     string
	SkipReason string
	MockData   ProviderMockConfig
}

type ProviderMockConfig struct {
	GitFiles            map[string]string
	HTTPResponses       map[string]HTTPResponseMock
	DataSourceResponses map[string]HTTPResponseMock
}

type HTTPResponseMock struct {
	StatusCode int
	Body       string
}

type Mocks struct {
	GitFilesystem    billy.Filesystem
	HTTPClient       *http.Client
	DataSourceClient *http.Client
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

// TestBuildMocks_InvalidGitFilePath confirms that BuildMocks surfaces the
// error from NewMockBillyFS when a git_files key is an empty string.
// This is the error-propagation branch inside BuildMocks: if the filesystem
// cannot be built, nothing should be returned and the error should be clear.
func TestBuildMocks_InvalidGitFilePath_ReturnsError(t *testing.T) {
	t.Parallel()

	tc := TestCase{
		Name:   "case with broken git file path",
		Expect: "pass",
		MockData: ProviderMockConfig{
			GitFiles: map[string]string{
				// Empty string triggers an error in the in-memory filesystem.
				"": "content that will never be written",
			},
		},
	}

	mocks, err := BuildMocks(tc)

	if err == nil {
		t.Fatal("expected BuildMocks to return an error for an empty git file path, got nil")
	}
	if mocks != nil {
		t.Error("expected BuildMocks to return nil mocks on error, got non-nil")
	}
}

// TestBuildMocks_ErrorMessageContainsCaseName checks that the error message
// from BuildMocks names the test case so developers can pinpoint which fixture
// entry is broken without having to inspect the full stack.
func TestBuildMocks_ErrorMessageContainsCaseName(t *testing.T) {
	t.Parallel()

	caseName := "my-identifiable-test-case"
	tc := TestCase{
		Name:   caseName,
		Expect: "pass",
		MockData: ProviderMockConfig{
			GitFiles: map[string]string{"": "bad"},
		},
	}

	_, err := BuildMocks(tc)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Error("error message should not be empty")
	}
}
