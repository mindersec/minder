// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestMockHTTPHandler(t *testing.T) {
	t.Parallel()

	// Setup a mock dictionary
	dict := starlark.NewDict(3)

	resp1 := &MockResponse{
		Body:       "ok",
		StatusCode: 200,
	}
	resp2 := &MockResponse{
		Body:       `{"error": "not found"}`,
		StatusCode: 404,
	}
	resp3 := &MockResponse{
		Body:       "wildcard matched",
		StatusCode: 201,
	}

	require.NoError(t, dict.SetKey(starlark.String("/api/v1/exact"), resp1))
	require.NoError(t, dict.SetKey(starlark.String("/api/v1/missing"), resp2))
	require.NoError(t, dict.SetKey(starlark.String("/api/v1/users/{user}/details"), resp3))

	handler, err := buildMockHTTPHandler(dict)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Test 1: Exact match
	req1, _ := http.NewRequest(http.MethodGet, "/api/v1/exact", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	res1 := rec1.Result()
	assert.Equal(t, 200, res1.StatusCode)
	body1, _ := io.ReadAll(res1.Body)
	assert.Equal(t, "ok", string(body1))

	// Test 2: Another exact match
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/missing", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	res2 := rec2.Result()
	assert.Equal(t, 404, res2.StatusCode)
	body2, _ := io.ReadAll(res2.Body)
	assert.Equal(t, `{"error": "not found"}`, string(body2))

	// Test 3: Pattern match
	req3, _ := http.NewRequest(http.MethodGet, "/api/v1/users/john_doe/details", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	res3 := rec3.Result()
	assert.Equal(t, 201, res3.StatusCode)
	body3, _ := io.ReadAll(res3.Body)
	assert.Equal(t, "wildcard matched", string(body3))

	// Test 4: Unmatched URL
	req4, _ := http.NewRequest(http.MethodGet, "/api/v1/users/john_doe/other", nil)
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req4)
	res4 := rec4.Result()
	assert.Equal(t, 404, res4.StatusCode)

	// Test 5: Nil dictionary
	nilHandler, err := buildMockHTTPHandler(nil)
	require.NoError(t, err)
	assert.NotNil(t, nilHandler)
}

func TestMockResponseMethods(t *testing.T) {
	t.Parallel()

	resp := &MockResponse{}

	// Test body method
	bodyRes, err := resp.Attr("body")
	require.NoError(t, err)
	bodyBuiltin, ok := bodyRes.(*starlark.Builtin)
	require.True(t, ok)

	resVal, err := bodyBuiltin.CallInternal(nil, starlark.Tuple{starlark.String("hello")}, nil)
	require.NoError(t, err)
	newResp := resVal.(*MockResponse)
	assert.Equal(t, "hello", newResp.Body)
	assert.Equal(t, "hello", resp.Body) // original modified

	// Test code method
	codeRes, err := resp.Attr("code")
	require.NoError(t, err)
	codeBuiltin, ok := codeRes.(*starlark.Builtin)
	require.True(t, ok)

	resVal, err = codeBuiltin.CallInternal(nil, starlark.Tuple{starlark.MakeInt(400)}, nil)
	require.NoError(t, err)
	newResp = resVal.(*MockResponse)
	assert.Equal(t, 400, newResp.StatusCode)
	assert.Equal(t, 400, resp.StatusCode) // original modified
}
