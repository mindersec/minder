// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gobwas/glob"
)

// HTTPMockResponse represents a mocked HTTP response configuration.
type HTTPMockResponse struct {
	StatusCode int
	Body       string
}

type mockEntry struct {
	pattern glob.Glob
	resp    *HTTPMockResponse
}

// MockRoundTripper intercepts HTTP requests and returns mocked responses based on URL glob patterns.
type MockRoundTripper struct {
	entries []mockEntry
}

// NewMockRoundTripper creates a new MockRoundTripper.
func NewMockRoundTripper() *MockRoundTripper {
	return &MockRoundTripper{}
}

// Add registers a new mock response for a given glob pattern.
func (m *MockRoundTripper) Add(pattern string, resp *HTTPMockResponse) error {
	g, err := glob.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	m.entries = append(m.entries, mockEntry{
		pattern: g,
		resp:    resp,
	})
	return nil
}

// RoundTrip executes the round trip, returning a mocked response if a match is found.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	reqURL := req.URL.String()

	for _, entry := range m.entries {
		if entry.pattern.Match(reqURL) {
			resp := &http.Response{
				StatusCode: entry.resp.StatusCode,
				Body:       io.NopCloser(bytes.NewBufferString(entry.resp.Body)),
				Header:     make(http.Header),
				Request:    req,
			}
			return resp, nil
		}
	}

	return nil, fmt.Errorf("unmatched URL: %s", reqURL)
}
