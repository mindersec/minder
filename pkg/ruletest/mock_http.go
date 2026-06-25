// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"errors"
	"fmt"
	"net/http"

	"go.starlark.net/starlark"
)

// MockResponse represents a mocked HTTP response in Starlark.
// It is the expected value type for the dictionary provided to the
// mock_http parameter in eval().
type MockResponse struct {
	StatusCode int
	Body       string
}

// Ensure MockResponse implements starlark.Value and starlark.HasAttrs
var (
	_ starlark.Value    = (*MockResponse)(nil)
	_ starlark.HasAttrs = (*MockResponse)(nil)
)

func (m *MockResponse) String() string {
	return fmt.Sprintf("mock_response(code=%d, body=%q)", m.StatusCode, m.Body)
}

// Type returns the Starlark type name.
func (*MockResponse) Type() string {
	return "mock_response"
}

// Freeze makes the value immutable.
func (*MockResponse) Freeze() {}

// Truth returns the truth value of the mock response.
func (*MockResponse) Truth() starlark.Bool {
	return starlark.True
}

// Hash returns a hash value for the mock response.
func (*MockResponse) Hash() (uint32, error) {
	return 0, errors.New("unhashable type: mock_response")
}

// Attr retrieves the attribute for Starlark.
func (m *MockResponse) Attr(name string) (starlark.Value, error) {
	switch name {
	case "code":
		return starlark.NewBuiltin("code", m.builtinCode), nil
	case "body":
		return starlark.NewBuiltin("body", m.builtinBody), nil
	default:
		return nil, nil
	}
}

// AttrNames returns the list of attribute names.
func (*MockResponse) AttrNames() []string {
	return []string{"body", "code"}
}

func (m *MockResponse) builtinCode(
	_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var code int
	if err := starlark.UnpackArgs("code", args, kwargs, "status", &code); err != nil {
		return nil, err
	}

	m.StatusCode = code
	return m, nil
}

func (m *MockResponse) builtinBody(
	_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var payload string
	if err := starlark.UnpackArgs("body", args, kwargs, "payload", &payload); err != nil {
		return nil, err
	}

	m.Body = payload
	return m, nil
}

func builtinCode(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var code int
	if err := starlark.UnpackArgs("code", args, kwargs, "status", &code); err != nil {
		return nil, err
	}
	return &MockResponse{
		StatusCode: code,
	}, nil
}

func builtinBody(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var payload string
	if err := starlark.UnpackArgs("body", args, kwargs, "payload", &payload); err != nil {
		return nil, err
	}
	return &MockResponse{
		StatusCode: 200, // default code
		Body:       payload,
	}, nil
}

// buildMockHTTPHandler creates a new http.Handler from a Starlark dictionary.
// The dictionary is expected to have string keys (representing the HTTP URL pattern)
// and mock_response values (created via body() or code() built-ins).
func buildMockHTTPHandler(mockDict *starlark.Dict) (http.Handler, error) {
	mux := http.NewServeMux()

	if mockDict == nil {
		return mux, nil
	}

	for _, key := range mockDict.Keys() {
		val, found, err := mockDict.Get(key)
		if err != nil || !found {
			continue // Should not happen
		}

		keyStr, ok := starlark.AsString(key)
		if !ok {
			return nil, fmt.Errorf("mock endpoints must have string keys, got %s", key.Type())
		}

		mockResp, ok := val.(*MockResponse)
		if !ok {
			return nil, fmt.Errorf("mock endpoint %q must be mapped to a mock_response, got %s", keyStr, val.Type())
		}

		mux.HandleFunc(keyStr, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(mockResp.StatusCode)
			_, _ = w.Write([]byte(mockResp.Body))
		})
	}

	return mux, nil
}
