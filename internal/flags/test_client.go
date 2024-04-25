//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

var _ openfeature.IClient = (*FakeClient)(nil)

// AddHandler implements openfeature.IClient.
func (_ *FakeClient) AddHandler(_ openfeature.EventType, _ openfeature.EventCallback) {
	panic("unimplemented")
}

// AddHooks implements openfeature.IClient.
func (_ *FakeClient) AddHooks(_ ...openfeature.Hook) {
	panic("unimplemented")
}

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
	return defaultValue, errors.New("Not found")
}

// BooleanValueDetails implements openfeature.IClient.
func (_ *FakeClient) BooleanValueDetails(_ context.Context, _ string, _ bool,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (openfeature.BooleanEvaluationDetails, error) {
	panic("unimplemented")
}

// EvaluationContext implements openfeature.IClient.
func (_ *FakeClient) EvaluationContext() openfeature.EvaluationContext {
	panic("unimplemented")
}

// Float implements openfeature.IClient.
func (f *FakeClient) Float(_ context.Context, flag string, defaultValue float64,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) float64 {
	if v, ok := f.Data[flag]; ok {
		return v.(float64)
	}
	return defaultValue
}

// FloatValue implements openfeature.IClient.
func (f *FakeClient) FloatValue(_ context.Context, flag string, defaultValue float64,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (float64, error) {
	if v, ok := f.Data[flag]; ok {
		return v.(float64), nil
	}
	return defaultValue, errors.New("Not found")
}

// FloatValueDetails implements openfeature.IClient.
func (_ *FakeClient) FloatValueDetails(_ context.Context, _ string, _ float64,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (openfeature.FloatEvaluationDetails, error) {
	panic("unimplemented")
}

// Int implements openfeature.IClient.
func (f *FakeClient) Int(_ context.Context, flag string, defaultValue int64,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) int64 {
	if v, ok := f.Data[flag]; ok {
		return v.(int64)
	}
	return defaultValue
}

// IntValue implements openfeature.IClient.
func (f *FakeClient) IntValue(_ context.Context, flag string, defaultValue int64,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (int64, error) {
	if v, ok := f.Data[flag]; ok {
		return v.(int64), nil
	}
	return defaultValue, errors.New("Not found")
}

// IntValueDetails implements openfeature.IClient.
func (_ *FakeClient) IntValueDetails(_ context.Context, _ string, _ int64,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (openfeature.IntEvaluationDetails, error) {
	panic("unimplemented")
}

// Metadata implements openfeature.IClient.
func (_ *FakeClient) Metadata() openfeature.ClientMetadata {
	panic("unimplemented")
}

// Object implements openfeature.IClient.
func (f *FakeClient) Object(_ context.Context, flag string, defaultValue interface{},
	_ openfeature.EvaluationContext, _ ...openfeature.Option) interface{} {
	if v, ok := f.Data[flag]; ok {
		return v
	}
	return defaultValue
}

// ObjectValue implements openfeature.IClient.
func (f *FakeClient) ObjectValue(_ context.Context, flag string, defaultValue interface{},
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (interface{}, error) {
	if v, ok := f.Data[flag]; ok {
		return v, nil
	}
	return defaultValue, errors.New("Not found")
}

// ObjectValueDetails implements openfeature.IClient.
func (_ *FakeClient) ObjectValueDetails(_ context.Context, _ string, _ interface{},
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (openfeature.InterfaceEvaluationDetails, error) {
	panic("unimplemented")
}

// RemoveHandler implements openfeature.IClient.
func (_ *FakeClient) RemoveHandler(_ openfeature.EventType, _ openfeature.EventCallback) {
	panic("unimplemented")
}

// SetEvaluationContext implements openfeature.IClient.
func (_ *FakeClient) SetEvaluationContext(_ openfeature.EvaluationContext) {
	panic("unimplemented")
}

// String implements openfeature.IClient.
func (f *FakeClient) String(_ context.Context, flag string, defaultValue string,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) string {
	if v, ok := f.Data[flag]; ok {
		return v.(string)
	}
	return defaultValue
}

// StringValue implements openfeature.IClient.
func (f *FakeClient) StringValue(_ context.Context, flag string, defaultValue string,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (string, error) {
	if v, ok := f.Data[flag]; ok {
		return v.(string), nil
	}
	return defaultValue, errors.New("Not found")
}

// StringValueDetails implements openfeature.IClient.
func (_ *FakeClient) StringValueDetails(_ context.Context, _ string, _ string,
	_ openfeature.EvaluationContext, _ ...openfeature.Option) (openfeature.StringEvaluationDetails, error) {
	panic("unimplemented")
}
