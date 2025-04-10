// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
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

			props := NewProperties(input)

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

			props := NewProperties(input)

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
		"id":          1,
		"is_private":  true,
		"count":       int64(5),
		"large":       int64(math.MaxInt64),
		"from_string": "2",
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
			errString: "failed to get int64 value",
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
		{
			name:     "AsUint64 from int",
			propName: "id",
			expValue: 1,
		},
		{
			name:     "AsUint64 from int",
			propName: "id",
			expValue: 1,
			callGet:  true,
		},
		{
			name:     "AsUint64 from string",
			propName: "from_string",
			expValue: 2,
		},
		{
			name:     "AsUint64 from string",
			propName: "from_string",
			expValue: 2,
			callGet:  true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			props := NewProperties(input)

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
		"id":          1,
		"is_private":  true,
		"count":       uint64(5),
		"large":       uint64(math.MaxUint64),
		"from_string": "2",
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
			errString: "failed to get uint64 value",
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
		{
			name:     "AsUint64 from int",
			propName: "id",
			expValue: 1,
		},
		{
			name:     "AsUint64 from int",
			propName: "id",
			expValue: 1,
			callGet:  true,
		},
		{
			name:     "AsUint64 from string",
			propName: "from_string",
			expValue: 2,
		},
		{
			name:     "AsUint64 from string",
			propName: "from_string",
			expValue: 2,
			callGet:  true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			props := NewProperties(input)

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

		props := NewProperties(nil)
		require.NotNil(t, props)
		p := props.GetProperty("test")
		require.Nil(t, p)
	})

	t.Run("reserved key", func(t *testing.T) {
		t.Parallel()

		testKey := internalPrefix + "test"

		_ = NewProperties(map[string]any{
			testKey: true,
		})
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

	props := NewProperties(input)

	count := 0
	for key, p := range props.Iterate() {
		count++
		output[key] = p.RawValue()
	}
	require.Equal(t, input, output)
	require.Equal(t, 2, count)
}

func TestMerge(t *testing.T) {
	t.Parallel()

	t.Run("merge two props", func(t *testing.T) {
		t.Parallel()

		props1 := NewProperties(map[string]any{
			"name": "test",
		})

		props2 := NewProperties(map[string]any{
			"is_private": true,
		})

		merged := props1.Merge(props2)

		expected := map[string]any{
			"name":       "test",
			"is_private": true,
		}

		output := make(map[string]any)
		for key, p := range merged.Iterate() {
			output[key] = p.RawValue()
		}
		require.Equal(t, expected, output)
	})

	t.Run("other is nil", func(t *testing.T) {
		t.Parallel()

		props1 := NewProperties(map[string]any{
			"name": "test",
		})

		merged := props1.Merge(nil)

		expected := map[string]any{
			"name": "test",
		}

		output := make(map[string]any)
		for key, p := range merged.Iterate() {
			output[key] = p.RawValue()
		}
		require.Equal(t, expected, output)
	})

	t.Run("self is nil", func(t *testing.T) {
		t.Parallel()

		props2 := NewProperties(map[string]any{
			"is_private": true,
		})

		var nilP *Properties
		merged := nilP.Merge(props2)

		expected := map[string]any{
			"is_private": true,
		}

		output := make(map[string]any)
		for key, p := range merged.Iterate() {
			output[key] = p.RawValue()
		}
		require.Equal(t, expected, output)
	})
}

func TestFilteredCopy(t *testing.T) {
	t.Parallel()

	t.Run("filter one", func(t *testing.T) {
		t.Parallel()

		props := NewProperties(map[string]any{
			"name":       "test",
			"is_private": true,
		})

		filter := func(key string, _ *Property) bool {
			return key == "name"
		}

		filtered := props.FilteredCopy(filter)

		expected := map[string]any{
			"name": "test",
		}

		output := make(map[string]any)
		for key, p := range filtered.Iterate() {
			output[key] = p.RawValue()
		}
		require.Equal(t, expected, output)
	})
}

func TestProperties_ToProtoStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		props    map[string]any
		expected *structpb.Struct
	}{
		{
			name: "mixed types",
			props: map[string]any{
				"string": "value",
				"int":    42,
				"bool":   true,
				"float":  3.14,
			},
			expected: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"string": structpb.NewStringValue("value"),
					"int":    structpb.NewNumberValue(42),
					"bool":   structpb.NewBoolValue(true),
					"float":  structpb.NewNumberValue(3.14),
				},
			},
		},
		{
			name:     "empty properties",
			props:    map[string]any{},
			expected: &structpb.Struct{Fields: map[string]*structpb.Value{}},
		},
		{
			name:     "nil properties",
			props:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var p *Properties
			if tt.props != nil {
				p = NewProperties(tt.props)
			}

			result := p.ToProtoStruct()

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.expected.Fields), len(result.Fields))
				for key, expectedValue := range tt.expected.Fields {
					assert.Contains(t, result.Fields, key)
					assert.Equal(t, expectedValue.GetKind(), result.Fields[key].GetKind())
					assert.Equal(t, expectedValue.AsInterface(), result.Fields[key].AsInterface())
				}
			}
		})
	}
}

func TestNewPropertiesWithSkipPrefixCheck(t *testing.T) {
	t.Parallel()

	// Test case with reserved prefix, without skip option
	reservedProps := map[string]any{
		"minder.internal.test": "value",
	}

	// Test case with reserved prefix, with skip option
	props := NewProperties(reservedProps, withSkipPrefixCheckTestOnly())

	// Verify the property was actually added
	prop := props.GetProperty("minder.internal.test")
	if prop == nil {
		t.Error("Expected property to be present")
	}
	if val := prop.GetString(); val != "value" {
		t.Errorf("Expected value 'value', got '%s'", val)
	}
}

// withSkipPrefixCheckTestOnly returns an option to skip checking the prefix
// This should only be used for testing purposes
func withSkipPrefixCheckTestOnly() newPropertiesOption {
	return func(c *newPropertiesConfig) {
		c.skipPrefixCheck = true
	}
}

func TestProperties_SetKeyValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		value   any
		wantErr bool
	}{
		{
			name:    "Set string value",
			key:     "testKey",
			value:   "testValue",
			wantErr: false,
		},
		{
			name:    "Set int64 value",
			key:     "intKey",
			value:   int64(42),
			wantErr: false,
		},
		{
			name:    "Set uint64 value",
			key:     "uintKey",
			value:   uint64(42),
			wantErr: false,
		},
		{
			name:    "Set bool value",
			key:     "boolKey",
			value:   true,
			wantErr: false,
		},
		{
			name:    "Set invalid value",
			key:     "invalidKey",
			value:   complex(1, 2),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewProperties(map[string]any{}, withSkipPrefixCheckTestOnly())
			err := p.SetKeyValue(tt.key, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetKeyValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				prop := p.GetProperty(tt.key)
				if prop == nil {
					t.Errorf("Property not set for key %s", tt.key)
					return
				}

				switch v := tt.value.(type) {
				case string:
					if got := prop.GetString(); got != v {
						t.Errorf("Expected string value %v, got %v", v, got)
					}
				case int64:
					if got := prop.GetInt64(); got != v {
						t.Errorf("Expected int64 value %v, got %v", v, got)
					}
				case uint64:
					if got := prop.GetUint64(); got != v {
						t.Errorf("Expected uint64 value %v, got %v", v, got)
					}
				case bool:
					if got := prop.GetBool(); got != v {
						t.Errorf("Expected bool value %v, got %v", v, got)
					}
				}
			}
		})
	}
}

func TestProperties_ToLogDict(t *testing.T) {
	t.Parallel()

	props := NewProperties(map[string]any{
		"string": "test",
		"int":    42,
		"bool":   true,
		"float":  3.14,
	})
	require.NotNil(t, props)

	dict := props.ToLogDict()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	logger.Info().Dict("properties", dict).Msg("Test log")

	var result map[string]any
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Check if the properties are correctly logged
	properties, ok := result["properties"].(map[string]any)
	require.True(t, ok, "couldn't convert to map[string]any")

	expectedProps := map[string]interface{}{
		"string": "test",
		"int":    float64(42), // JSON numbers are floats
		"bool":   true,
		"float":  3.14,
	}

	for key, expectedValue := range expectedProps {
		actualValue, exists := properties[key]
		require.True(t, exists, "property %s not found in log output", key)
		require.Equal(t, expectedValue, actualValue)
	}
}

func TestProperties_Len(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    *Properties
		want int
	}{
		{
			name: "nil Properties",
			p:    nil,
			want: 0,
		},
		{
			name: "empty Properties",
			p: func() *Properties {
				p := NewProperties(map[string]any{})
				return p
			}(),
			want: 0,
		},
		{
			name: "Properties with one item",
			p: func() *Properties {
				p := NewProperties(map[string]any{"key1": "value1"})
				return p
			}(),
			want: 1,
		},
		{
			name: "Properties with multiple items",
			p: func() *Properties {
				p := NewProperties(map[string]any{
					"key1": "value1",
					"key2": 42,
					"key3": true,
				})
				return p
			}(),
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.p.Len(); got != tt.want {
				t.Errorf("Properties.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Equal(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name  string
		a     any
		b     any
		equal bool
	}{
		{
			name:  "equal string",
			a:     "test",
			b:     "test",
			equal: true,
		},
		{
			name:  "different string",
			a:     "test",
			b:     "test2",
			equal: false,
		},
		{
			name:  "equal int64",
			a:     int64(42),
			b:     int64(42),
			equal: true,
		},
		{
			name:  "different int64",
			a:     int64(42),
			b:     int64(43),
			equal: false,
		},
		{
			name:  "equal uint64",
			a:     uint64(42),
			b:     uint64(42),
			equal: true,
		},
		{
			name:  "different uint64",
			a:     uint64(42),
			b:     uint64(43),
			equal: false,
		},
		{
			name:  "equal bool",
			a:     true,
			b:     true,
			equal: true,
		},
		{
			name:  "different bool",
			a:     true,
			b:     false,
			equal: false,
		},
		{
			name:  "equal float64",
			a:     3.14,
			b:     3.14,
			equal: true,
		},
		{
			name:  "different float64",
			a:     3.14,
			b:     3.15,
			equal: false,
		},
		{
			name:  "equal map",
			a:     map[string]any{"test": "value"},
			b:     map[string]any{"test": "value"},
			equal: true,
		},
		{
			name:  "different map",
			a:     map[string]any{"test": "value"},
			b:     map[string]any{"test": "value2"},
			equal: false,
		},
		{
			name:  "equal nil",
			a:     nil,
			b:     nil,
			equal: true,
		},
		{
			name:  "different nil",
			a:     nil,
			b:     "test",
			equal: false,
		},
		{
			name:  "different types",
			a:     "test",
			b:     42,
			equal: false,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			p1, err := NewProperty(s.a)
			require.NoError(t, err)
			p2, err := NewProperty(s.b)
			require.NoError(t, err)

			assert.Equal(t, s.equal, p1.Equal(p2))
		})
	}
}

func TestProperties_Equal_Nils(t *testing.T) {
	t.Parallel()

	t.Run("both nil", func(t *testing.T) {
		t.Parallel()

		var p1, p2 *Property
		assert.True(t, p1.Equal(p2))
	})

	t.Run("nil parameter", func(t *testing.T) {
		t.Parallel()

		p1, err := NewProperty("test")
		require.NoError(t, err)
		var p2 *Property
		assert.False(t, p1.Equal(p2))
	})

	t.Run("nil receiver", func(t *testing.T) {
		t.Parallel()

		var p1 *Property
		p2, err := NewProperty("test")
		require.NoError(t, err)
		assert.False(t, p1.Equal(p2))
	})
}
