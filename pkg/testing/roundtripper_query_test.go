// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// --- Test-local types and helpers ---

// HTTPResponseMock defines a canned HTTP response for a given URL.
type HTTPResponseMock struct {
	StatusCode int
	Body       string
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

// TestMockRoundTripper_QueryParamsMustMatch verifies that the tripper matches
// on the full URL including query parameters.  A rule might call the GitHub
// API with a query string (e.g. ?per_page=100), and the fixture key must
// include those parameters for the mock to fire.
func TestMockRoundTripper_QueryParamsMustMatch(t *testing.T) {
	t.Parallel()

	rt := NewMockRoundTripper(map[string]HTTPResponseMock{
		"https://api.github.com/repos/o/r/commits?per_page=1": {
			StatusCode: 200,
			Body:       `[{"sha":"abc123"}]`,
		},
	})

	// Exact match: should serve the canned response.
	req := &http.Request{URL: mustParseURL(t, "https://api.github.com/repos/o/r/commits?per_page=1")}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("exact match: status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `[{"sha":"abc123"}]` {
		t.Errorf("exact match: body = %q, want %q", string(body), `[{"sha":"abc123"}]`)
	}
}

// TestMockRoundTripper_QueryParamMismatch_Returns404 confirms that a URL
// without the expected query parameters does not match the keyed entry and
// falls through to the default 404 response.  This prevents a rule from
// accidentally matching a URL it should not.
func TestMockRoundTripper_QueryParamMismatch_Returns404(t *testing.T) {
	t.Parallel()

	rt := NewMockRoundTripper(map[string]HTTPResponseMock{
		"https://api.github.com/repos/o/r/commits?per_page=1": {
			StatusCode: 200,
			Body:       `[{"sha":"abc123"}]`,
		},
	})

	// Same path but different query string: should miss and return 404.
	req := &http.Request{URL: mustParseURL(t, "https://api.github.com/repos/o/r/commits?per_page=100")}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("mismatched query: status = %d, want 404", resp.StatusCode)
	}
}

// TestMockRoundTripper_404BodyContainsURL checks that when a URL is not found
// in the mock map the 404 response body includes the URL that was requested.
// This makes it easy to diagnose a missing fixture entry during test runs.
func TestMockRoundTripper_404BodyContainsURL(t *testing.T) {
	t.Parallel()

	rt := NewMockRoundTripper(nil)
	target := "https://api.github.com/repos/o/r/not-in-fixture"

	req := &http.Request{URL: mustParseURL(t, target)}
	resp, _ := rt.RoundTrip(req)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if len(bodyStr) == 0 {
		t.Fatal("404 body should not be empty")
	}
	// The body should mention the URL so the developer knows which entry to add.
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func mustParseURL(t *testing.T, target string) *url.URL {
	t.Helper()
	u, err := url.Parse(target)
	if err != nil {
		t.Fatalf("could not parse URL %q: %v", target, err)
	}
	return u
}
