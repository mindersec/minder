// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockRoundTripper(t *testing.T) {
	t.Parallel()

	rt := NewMockRoundTripper()

	resp1 := &HTTPMockResponse{
		Body:       "ok",
		StatusCode: 200,
	}
	resp2 := &HTTPMockResponse{
		Body:       `{"error": "not found"}`,
		StatusCode: 404,
	}
	resp3 := &HTTPMockResponse{
		Body:       "wildcard matched",
		StatusCode: 201,
	}

	require.NoError(t, rt.Add("/api/v1/exact", resp1))
	require.NoError(t, rt.Add("/api/v1/missing", resp2))
	require.NoError(t, rt.Add("/api/v1/users/*/details", resp3))

	// Test 1: Exact match
	req1, _ := http.NewRequest(http.MethodGet, "/api/v1/exact", nil)
	res1, err := rt.RoundTrip(req1)
	require.NoError(t, err)
	assert.Equal(t, 200, res1.StatusCode)
	body1, _ := io.ReadAll(res1.Body)
	assert.Equal(t, "ok", string(body1))

	// Test 2: Another exact match
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/missing", nil)
	res2, err := rt.RoundTrip(req2)
	require.NoError(t, err)
	assert.Equal(t, 404, res2.StatusCode)
	body2, _ := io.ReadAll(res2.Body)
	assert.Equal(t, `{"error": "not found"}`, string(body2))

	// Test 3: Glob wildcard match
	req3, _ := http.NewRequest(http.MethodGet, "/api/v1/users/john_doe/details", nil)
	res3, err := rt.RoundTrip(req3)
	require.NoError(t, err)
	assert.Equal(t, 201, res3.StatusCode)
	body3, _ := io.ReadAll(res3.Body)
	assert.Equal(t, "wildcard matched", string(body3))

	// Test 4: Unmatched URL
	req4, _ := http.NewRequest(http.MethodGet, "/api/v1/users/john_doe/other", nil)
	_, err = rt.RoundTrip(req4)
	require.ErrorContains(t, err, "unmatched URL: /api/v1/users/john_doe/other")

	// Test 5: Empty configuration
	rtEmpty := NewMockRoundTripper()
	_, err = rtEmpty.RoundTrip(req1)
	require.ErrorContains(t, err, "unmatched URL")
}
