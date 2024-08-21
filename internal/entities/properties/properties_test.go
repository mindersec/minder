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

package properties

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoolGetters(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":         1,
		"is_private": true,
	}

	scenarios := []struct {
		name      string
		propName  string
		errString string
		expValue  bool
		callGet   bool
	}{
		{
			name:     "AsBool known property",
			propName: "is_private",
			expValue: true,
		},
		{
			name:     "GetBool known property",
			propName: "is_private",
			expValue: true,
			callGet:  true,
		},
		{
			name:      "AsBool unknown property",
			propName:  "unknown",
			errString: "property is nil",
		},
		{
			name:     "GetBool unknown property",
			propName: "unknown",
			expValue: false,
			callGet:  true,
		},
		{
			name:      "AsBool non-bool property",
			propName:  "id",
			errString: "value is not of type bool",
		},
		{
			name:     "GetBool non-bool property",
			propName: "id",
			expValue: false,
			callGet:  true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			props, err := NewProperties(input)
			require.NoError(t, err)

			p := props.GetProperty(s.propName)
			if s.callGet {
				got := p.GetBool()
				require.Equal(t, s.expValue, got)
			} else {
				got, err := p.AsBool()
				if s.errString == "" {
					require.NoError(t, err)
					require.Equal(t, s.expValue, got)
				} else {
					require.Error(t, err)
					require.ErrorContains(t, err, s.errString)
				}
			}
		})
	}
}

func TestStringGetters(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":         1,
		"is_private": true,
		"name":       "test",
	}

	scenarios := []struct {
		name      string
		propName  string
		errString string
		expValue  string
		callGet   bool
	}{
		{
			name:     "AsString known property",
			propName: "name",
			expValue: "test",
		},
		{
			name:     "GetString known property",
			propName: "name",
			expValue: "test",
			callGet:  true,
		},
		{
			name:      "AsString unknown property",
			propName:  "unknown",
			errString: "property is nil",
		},
		{
			name:     "GetString unknown property",
			propName: "unknown",
			expValue: "",
			callGet:  true,
		},
		{
			name:      "AsString non-string property",
			propName:  "id",
			errString: "value is not of type string",
		},
		{
			name:     "GetString non-string property",
			propName: "id",
			expValue: "",
			callGet:  true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			props, err := NewProperties(input)
			require.NoError(t, err)

			p := props.GetProperty(s.propName)
			if s.callGet {
				got := p.GetString()
				require.Equal(t, s.expValue, got)
			} else {
				got, err := p.AsString()
				if s.errString == "" {
					require.NoError(t, err)
					require.Equal(t, s.expValue, got)
				} else {
					require.Error(t, err)
					require.ErrorContains(t, err, s.errString)
				}
			}
		})
	}
}

func TestInt64Getters(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":         1,
		"is_private": true,
		"count":      int64(5),
		"large":      int64(math.MaxInt64),
	}

	scenarios := []struct {
		name      string
		propName  string
		errString string
		expValue  int64
		callGet   bool
	}{
		{
			name:     "AsInt64 known property",
			propName: "count",
			expValue: 5,
		},
		{
			name:     "GetInt64 known property",
			propName: "count",
			expValue: 5,
			callGet:  true,
		},
		{
			name:      "AsInt64 unknown property",
			propName:  "unknown",
			errString: "property is nil",
		},
		{
			name:     "GetInt64 unknown property",
			propName: "unknown",
			expValue: 0,
			callGet:  true,
		},
		{
			name:      "AsInt64 non-int64 property",
			propName:  "is_private",
			errString: "failed to get int64 value: value is not a map",
		},
		{
			name:     "GetInt64 non-int64 property",
			propName: "is_private",
			expValue: 0,
			callGet:  true,
		},
		{
			name:     "AsInt64 large property",
			propName: "large",
			expValue: math.MaxInt64,
		},
		{
			name:     "GetInt64 large property",
			propName: "large",
			expValue: math.MaxInt64,
			callGet:  true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			props, err := NewProperties(input)
			require.NoError(t, err)

			p := props.GetProperty(s.propName)
			if s.callGet {
				got := p.GetInt64()
				require.Equal(t, s.expValue, got)
			} else {
				got, err := p.AsInt64()
				if s.errString == "" {
					require.NoError(t, err)
					require.Equal(t, s.expValue, got)
				} else {
					require.Error(t, err)
					require.ErrorContains(t, err, s.errString)
				}
			}
		})
	}
}

func TestUint64Getters(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":         1,
		"is_private": true,
		"count":      uint64(5),
		"large":      uint64(math.MaxUint64),
	}

	scenarios := []struct {
		name      string
		propName  string
		errString string
		expValue  uint64
		callGet   bool
	}{
		{
			name:     "AsUint64 known property",
			propName: "count",
			expValue: 5,
		},
		{
			name:     "GetUint64 known property",
			propName: "count",
			expValue: 5,
			callGet:  true,
		},
		{
			name:      "AsUint64 unknown property",
			propName:  "unknown",
			errString: "property is nil",
		},
		{
			name:     "GetUint64 unknown property",
			propName: "unknown",
			expValue: 0,
			callGet:  true,
		},
		{
			name:      "AsUint64 non-uint64 property",
			propName:  "is_private",
			errString: "failed to get uint64 value: value is not a map",
		},
		{
			name:     "GetUint64 non-uint64 property",
			propName: "is_private",
			expValue: 0,
			callGet:  true,
		},
		{
			name:     "AsUint64 large property",
			propName: "large",
			expValue: math.MaxUint64,
		},
		{
			name:     "GetUint64 large property",
			propName: "large",
			expValue: math.MaxUint64,
			callGet:  true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			props, err := NewProperties(input)
			require.NoError(t, err)

			p := props.GetProperty(s.propName)
			if s.callGet {
				got := p.GetUint64()
				require.Equal(t, s.expValue, got)
			} else {
				got, err := p.AsUint64()
				if s.errString == "" {
					require.NoError(t, err)
					require.Equal(t, s.expValue, got)
				} else {
					require.Error(t, err)
					require.ErrorContains(t, err, s.errString)
				}
			}
		})
	}
}

func TestNewProperty(t *testing.T) {
	t.Parallel()

	p, err := NewProperty(true)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, true, p.GetBool())
}

func TestNewProperties(t *testing.T) {
	t.Parallel()

	t.Run("nil input", func(t *testing.T) {
		t.Parallel()

		props, err := NewProperties(nil)
		require.NoError(t, err)
		require.NotNil(t, props)
		p := props.GetProperty("test")
		require.Nil(t, p)
	})

	t.Run("reserved key", func(t *testing.T) {
		t.Parallel()

		testKey := internalPrefix + "test"

		props, err := NewProperties(map[string]any{
			testKey: true,
		})
		require.Contains(t, err.Error(), fmt.Sprintf("property key %s is reserved", testKey))
		require.Nil(t, props)
	})
}

func TestNilReceiver(t *testing.T) {
	t.Parallel()

	t.Run("GetProperty", func(t *testing.T) {
		t.Parallel()

		var ps *Properties
		p := ps.GetProperty("test")
		require.Nil(t, p)
		require.False(t, p.GetBool())
	})
}

func TestUnwrapTypeErrors(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name  string
		value any
		err   string
	}{
		{
			name: "no type field",
			value: map[string]any{
				valueKey: 1,
			},
			err: "type field not found",
		},
		{
			name: "unexpected type value",
			value: map[string]any{
				typeKey:  "unknown",
				valueKey: 1,
			},
			err: "value is not of type",
		},
		{
			name: "no value field",
			value: map[string]any{
				typeKey: typeInt64,
			},
			err: "value field not found",
		},
		{
			name: "invalid value type",
			value: map[string]any{
				typeKey:  typeInt64,
				valueKey: 1,
			},
			err: "invalid syntax",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			prop, err := NewProperty(s.value)
			require.NoError(t, err)
			// we test int64, but that's just a coincidence as it calls unwrapTypedValue internally
			_, err = prop.AsInt64()
			require.Contains(t, err.Error(), s.err)
		})
	}
}

func TestIterator(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"name":       "test",
		"is_private": true,
	}

	output := make(map[string]any)

	props, err := NewProperties(input)
	require.NoError(t, err)

	count := 0
	for key, p := range props.Iterate() {
		count++
		output[key] = p.RawValue()
	}
	require.Equal(t, input, output)
	require.Equal(t, 2, count)
}
