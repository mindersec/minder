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
		"id":         1,
		"is_private": true,
		"count":      int64(5),
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
			errString: "value is not of type int64",
		},
		{
			name:     "GetInt64 non-int64 property",
			propName: "is_private",
			expValue: 0,
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

func TestNewProperty(t *testing.T) {
	t.Parallel()

	p := NewProperty(true)
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
