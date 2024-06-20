//
// Copyright 2024 Stacklok, Inc.
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

package provider

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// RunUnitTestSuite runs the unit test suite.
func RunUnitTestSuite(t *testing.T) {
	t.Helper()

	suite.Run(t, new(UnitTestSuite))
}

// UnitTestSuite is the test suite for the unit tests.
type UnitTestSuite struct {
	suite.Suite
}

func ptr[T any](v T) *T {
	return &v
}

func (s *UnitTestSuite) TestParseConfigAttribute() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name      string
		attr      string
		attrName  *string
		attrValue *string
		err       bool
	}{
		{
			name:      "happy path",
			attr:      "foo.bar.baz=quux",
			attrName:  ptr("foo.bar.baz"),
			attrValue: ptr("quux"),
		},
		{
			name: "path only",
			attr: "foo.bar.baz",
			err:  true,
		},
		{
			name: "no value",
			attr: "foo.bar.baz=",
			err:  true,
		},
		{
			name: "no path",
			attr: "=quux",
			err:  true,
		},
		{
			name: "empty",
			err:  true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			attrName, attrValue, err := parseConfigAttribute(tt.attr)
			if tt.err {
				require.Error(t, err)
				require.Equal(t, "", attrName)
				require.Equal(t, "", attrValue)
			}
			if tt.attrName != nil {
				require.NoError(t, err)
				require.Equal(t, *tt.attrName, attrName)
				require.Equal(t, *tt.attrValue, attrValue)
			}
		})
	}
}

type AnotherJsonStruct struct {
	Tizio *string `json:"tizio,omitempty"`
}

type JsonStruct struct {
	Foo    string             `json:"foo,omitempty"`
	BarBaz string             `json:"bar_baz,omitempty"`
	Quux   *AnotherJsonStruct `json:"quux,omitempty"`
}

var emptyStructPtr = &JsonStruct{}

func (s *UnitTestSuite) TestByJSONName() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name      string
		input     reflect.Value
		fieldName string
		result    reflect.Value
		err       bool
	}{
		{
			name: "simple field",
			input: reflect.ValueOf(
				JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			),
			fieldName: "foo",
			result:    reflect.ValueOf("test1"),
		},
		{
			name: "complex field",
			input: reflect.ValueOf(
				JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			),
			fieldName: "bar_baz",
			result:    reflect.ValueOf("test2"),
		},
		{
			name: "no such field",
			input: reflect.ValueOf(
				JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			),
			fieldName: "whatever",
			err:       true,
		},
		{
			name:      "not a struct",
			input:     reflect.ValueOf("this is not a struct"),
			fieldName: "whatever",
			err:       true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := byJSONName(tt.input, tt.fieldName)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, res.IsValid())
				require.True(t, tt.result.Equal(res))
			}
		})
	}
}

func (s *UnitTestSuite) TestGetField() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name      string
		input     reflect.Value
		fieldName string
		result    reflect.Value
		err       bool
	}{
		// getField from structs
		{
			name: "simple struct field",
			input: reflect.ValueOf(
				JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			),
			fieldName: "foo",
			result:    reflect.ValueOf("test1"),
		},
		{
			name: "complex struct field",
			input: reflect.ValueOf(
				JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			),
			fieldName: "bar_baz",
			result:    reflect.ValueOf("test2"),
		},
		{
			name: "no such struct field",
			input: reflect.ValueOf(
				JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			),
			fieldName: "whatever",
			err:       true,
		},

		// getField from maps
		{
			name: "simple from map",
			input: reflect.ValueOf(map[string]any{
				"quux": JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			}),
			fieldName: "quux",
			result: reflect.ValueOf(
				JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
				},
			),
		},
		{
			name:      "missing from map",
			input:     reflect.ValueOf(map[string]JsonStruct{}),
			fieldName: "missing1",
			result:    reflect.ValueOf(*emptyStructPtr),
		},
		{
			name:      "pointer from map",
			input:     reflect.ValueOf(map[string]*JsonStruct{}),
			fieldName: "missing2",
			result:    reflect.ValueOf(emptyStructPtr),
		},

		// getField from scalar type
		{
			name:      "not a struct",
			input:     reflect.ValueOf("this is not a struct"),
			fieldName: "whatever",
			err:       true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := getField(tt.input, tt.fieldName)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, res.IsValid())

				//nolint:exhaustive
				switch tt.result.Kind() {
				case reflect.Pointer:
					require.Truef(t, tt.result.Elem().Equal(res.Elem()),
						"expected %v, got %v",
						tt.result.Elem(), res.Elem(),
					)
				default:
					require.Truef(t, tt.result.Equal(res),
						"expected %v, got %v",
						tt.result, res,
					)
				}
			}
		})
	}
}

func (s *UnitTestSuite) TestInitField() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name  string
		input func() reflect.Value
		err   bool
	}{
		// initField only initializes structs and maps to
		// prevent null pointer dereference.
		{
			name: "null struct ptr",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf((*JsonStruct)(nil))).Elem()
			},
		},
		{
			name: "scalar ptr",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf((*int)(nil))).Elem()
			},
		},
		{
			name: "scalar ptr bis",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf((*string)(nil))).Elem()
			},
		},
		{
			name: "map",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf(map[string]any{})).Elem()
			},
		},
		{
			name: "map ptr",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf((*map[string]any)(nil))).Elem()
			},
		},

		// everything else returns an error
		{
			name: "empty struct",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf(JsonStruct{})).Elem()
			},
			err: true,
		},
		{
			name: "scalar",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf(42)).Elem()
			},
			err: true,
		},
		{
			name: "scalar bis",
			input: func() reflect.Value {
				return reflect.New(reflect.TypeOf("")).Elem()
			},
			err: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// we instantiate a new one because initField
			// works by side effect
			actualInput := tt.input()
			err := initField(actualInput)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, actualInput.IsValid())
				require.True(t, actualInput.CanSet())
			}
		})
	}
}

func (s *UnitTestSuite) TestConfigAttribute() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name      string
		input     reflect.Value
		attrName  string
		attrValue *string
		checkFunc func(*testing.T, reflect.Value)
		err       bool
	}{
		// getField from structs
		{
			name: "no value unsets field",
			input: reflect.ValueOf(
				&JsonStruct{
					Foo:    "test1",
					BarBaz: "test2",
					Quux: &AnotherJsonStruct{
						Tizio: ptr("caio"),
					},
				},
			),
			attrName: "foo",
			checkFunc: func(t *testing.T, v reflect.Value) {
				t.Helper()
				res, ok := v.Interface().(*JsonStruct)
				require.True(t, ok)
				require.Equal(t, "", res.Foo)
				require.Equal(t, "test2", res.BarBaz)
				require.Equal(t, ptr("caio"), res.Quux.Tizio)
			},
		},
		{
			name:     "must pass by reference",
			input:    reflect.ValueOf(JsonStruct{}),
			attrName: "foo",
			err:      true,
		},
		{
			name: "recur into structure",
			input: reflect.ValueOf(&JsonStruct{
				Foo:    "test1",
				BarBaz: "test2",
				Quux: &AnotherJsonStruct{
					Tizio: ptr("caio"),
				},
			}),
			attrName: "quux.tizio",
			checkFunc: func(t *testing.T, v reflect.Value) {
				t.Helper()
				res, ok := v.Interface().(*JsonStruct)
				require.True(t, ok)
				require.Equal(t, "test1", res.Foo)
				require.Equal(t, "test2", res.BarBaz)
				require.Equal(t, "", *res.Quux.Tizio)
			},
		},
		{
			name: "path too short",
			input: reflect.ValueOf(&JsonStruct{
				Foo:    "test1",
				BarBaz: "test2",
				Quux: &AnotherJsonStruct{
					Tizio: ptr("caio"),
				},
			}),
			attrName: "quux",
			err:      true,
		},
		{
			name: "path too long",
			input: reflect.ValueOf(&JsonStruct{
				Foo:    "test1",
				BarBaz: "test2",
				Quux: &AnotherJsonStruct{
					Tizio: ptr("caio"),
				},
			}),
			attrName: "quux.tizio.wat",
			err:      true,
		},

		// changing values
		{
			name: "modify shallow",
			input: reflect.ValueOf(&JsonStruct{
				Foo:    "test1",
				BarBaz: "test2",
				Quux: &AnotherJsonStruct{
					Tizio: ptr("caio"),
				},
			}),
			attrName:  "foo",
			attrValue: ptr("modified"),
			checkFunc: func(t *testing.T, v reflect.Value) {
				t.Helper()
				res, ok := v.Interface().(*JsonStruct)
				require.True(t, ok)
				require.Equal(t, "modified", res.Foo)
				require.Equal(t, "test2", res.BarBaz)
				require.Equal(t, "caio", *res.Quux.Tizio)
			},
		},
		{
			name: "modify deep",
			input: reflect.ValueOf(&JsonStruct{
				Foo:    "test1",
				BarBaz: "test2",
				Quux: &AnotherJsonStruct{
					Tizio: ptr("caio"),
				},
			}),
			attrName:  "quux.tizio",
			attrValue: ptr("sempronio"),
			checkFunc: func(t *testing.T, v reflect.Value) {
				t.Helper()
				res, ok := v.Interface().(*JsonStruct)
				require.True(t, ok)
				require.Equal(t, "test1", res.Foo)
				require.Equal(t, "test2", res.BarBaz)
				require.Equal(t, "sempronio", *res.Quux.Tizio)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := configAttribute(tt.input, tt.attrName, tt.attrValue)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, tt.input)
				}
			}
		})
	}
}

func TestConfigReflection(t *testing.T) {
	t.Parallel()

	RunUnitTestSuite(t)
	// Call other test runner functions for additional test suites
}
