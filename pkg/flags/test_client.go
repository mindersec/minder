// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package flags

import (
	"context"
	"errors"

	"github.com/open-feature/go-sdk/openfeature"
)

// FakeClient implements a simple in-memory client for testing.
//
// see https://github.com/open-feature/go-sdk/issues/266 for the proper support.
type FakeClient struct {
	Data map[string]any
}

var _ Interface = (*FakeClient)(nil)

// Boolean implements openfeature.IClient.
func (f *FakeClient) Boolean(_ context.Context, flag string, defaultValue bool,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) bool {
	if v, ok := f.Data[flag]; ok {
		return v.(bool)
	}
	return defaultValue
}

// BooleanValue implements openfeature.IClient.
func (f *FakeClient) BooleanValue(_ context.Context, flag string, defaultValue bool,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (bool, error) {
	if v, ok := f.Data[flag]; ok {
		return v.(bool), nil
	}
	return defaultValue, errors.New("not found")
}
