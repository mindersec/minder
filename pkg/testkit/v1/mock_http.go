// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

// HTTPMockResponse represents a mocked HTTP response configuration.
type HTTPMockResponse struct {
	StatusCode int
	Body       string
}

// MockRoundTripper intercepts HTTP requests and returns mocked responses using an http.ServeMux.
type MockRoundTripper struct {
	mux *http.ServeMux
}

// NewMockRoundTripper creates a new MockRoundTripper.
func NewMockRoundTripper() *MockRoundTripper {
	return &MockRoundTripper{
		mux: http.NewServeMux(),
	}
}

// Add registers a new mock response for a given HTTP pattern.
func (m *MockRoundTripper) Add(pattern string, resp *HTTPMockResponse) error {
	m.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write([]byte(resp.Body))
	})
	return nil
}

// RoundTrip executes the round trip, returning a mocked response if a match is found.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	handler, pattern := m.mux.Handler(req)
	if pattern == "" {
		return nil, fmt.Errorf("unmatched URL: %s", req.URL.String())
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	resp.Request = req
	return resp, nil
}
