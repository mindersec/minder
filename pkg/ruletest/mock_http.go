// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"errors"
	"fmt"

	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
	"go.starlark.net/starlark"
)

// MockResponse represents a mocked HTTP response in Starlark.
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

func (m *MockResponse) Type() string {
	return "mock_response"
}

func (m *MockResponse) Freeze() {}

func (m *MockResponse) Truth() starlark.Bool {
	return starlark.True
}

func (m *MockResponse) Hash() (uint32, error) {
	return 0, errors.New("unhashable type: mock_response")
}

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

func (m *MockResponse) AttrNames() []string {
	return []string{"body", "code"}
}

func (m *MockResponse) builtinCode(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var code int
	if err := starlark.UnpackArgs("code", args, kwargs, "status", &code); err != nil {
		return nil, err
	}

	return &MockResponse{
		StatusCode: code,
		Body:       m.Body,
	}, nil
}

func (m *MockResponse) builtinBody(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var payload string
	if err := starlark.UnpackArgs("body", args, kwargs, "payload", &payload); err != nil {
		return nil, err
	}

	return &MockResponse{
		StatusCode: m.StatusCode,
		Body:       payload,
	}, nil
}

func builtinCode(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var code int
	if err := starlark.UnpackArgs("code", args, kwargs, "status", &code); err != nil {
		return nil, err
	}
	return &MockResponse{
		StatusCode: code,
	}, nil
}

func builtinBody(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var payload string
	if err := starlark.UnpackArgs("body", args, kwargs, "payload", &payload); err != nil {
		return nil, err
	}
	return &MockResponse{
		StatusCode: 200, // default code
		Body:       payload,
	}, nil
}

// NewMockRoundTripper creates a new tkv1.MockRoundTripper from a Starlark dictionary.
func NewMockRoundTripper(mockDict *starlark.Dict) (*tkv1.MockRoundTripper, error) {
	rt := tkv1.NewMockRoundTripper()

	if mockDict == nil {
		return rt, nil
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

		err = rt.Add(keyStr, &tkv1.HTTPMockResponse{
			StatusCode: mockResp.StatusCode,
			Body:       mockResp.Body,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to register mock endpoint %q: %w", keyStr, err)
		}
	}

	return rt, nil
}
