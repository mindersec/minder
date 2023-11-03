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

// Package util provides helper functions for the mediator CLI.
package util_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stacklok/minder/internal/util"
)

func TestJQReadFromAccessorString(t *testing.T) {
	t.Parallel()

	var want = "value"

	s, err := util.JQReadFrom[string](context.Background(), ".simple", map[string]any{
		"simple": want,
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.Equal(t, want, s, "Expected JQReadFrom() to return %v, got %v", want, s)
}

func TestJQReadFromAccessorNumber(t *testing.T) {
	t.Parallel()

	var want = 1

	n, err := util.JQReadFrom[int](context.Background(), ".number", map[string]any{
		"number": want,
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.Equal(t, want, n, "Expected JQReadFrom() to return %v, got %v", want, n)
}

func TestJQReadFromAccessorBoolean(t *testing.T) {
	t.Parallel()

	var want = true

	b, err := util.JQReadFrom[bool](context.Background(), ".boolean", map[string]any{
		"boolean": want,
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.Equal(t, want, b, "Expected JQReadFrom() to return %v, got %v", want, b)
}

func TestJQReadFromAccessorArray(t *testing.T) {
	t.Parallel()

	var want = []string{
		"one",
		"two",
		"three",
	}

	a, err := util.JQReadFrom[[]string](context.Background(), ".array", map[string]any{
		"array": []string{
			"one",
			"two",
			"three",
		},
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.Equal(t, want, a, "Expected JQReadFrom() to return %v, got %v", want, a)
}

func TestJQReadFromAccessorNestedArray(t *testing.T) {
	t.Parallel()

	var want = []string{
		"one",
		"two",
		"three",
	}

	a, err := util.JQReadFrom[[]string](context.Background(), ".nested.array", map[string]any{
		"nested": map[string]any{
			"array": []string{
				"one",
				"two",
				"three",
			},
		},
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.Equal(t, want, a, "Expected JQReadFrom() to return %v, got %v", want, a)
}

func TestJQReadFromAccessorObj(t *testing.T) {
	t.Parallel()

	var want = map[string]any{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	o, err := util.JQReadFrom[map[string]any](context.Background(), ".object", map[string]any{
		"object": map[string]any{
			"one":   1,
			"two":   2,
			"three": 3,
		},
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.True(t, reflect.DeepEqual(o, want), "Expected jQReadAsAny() to return %v, got %v", want, o)
}

func TestJQReadFromAccessorNestedObj(t *testing.T) {
	t.Parallel()

	var want = map[string]any{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	o, err := util.JQReadFrom[map[string]any](context.Background(), ".nested.object", map[string]any{
		"nested": map[string]any{
			"object": map[string]any{
				"one":   1,
				"two":   2,
				"three": 3,
			},
		},
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.True(t, reflect.DeepEqual(o, want), "Expected jQReadAsAny() to return %v, got %v", want, o)
}

func TestJQReadFromAccessorAny(t *testing.T) {
	t.Parallel()

	var want = map[string]any{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	o, err := util.JQReadFrom[any](context.Background(), ".nested.object", map[string]any{
		"nested": map[string]any{
			"object": map[string]any{
				"one":   1,
				"two":   2,
				"three": 3,
			},
		},
	})

	assert.NoError(t, err, "Unexpected error processing JQReadFrom()")
	assert.True(t, reflect.DeepEqual(o, want), "Expected jQReadAsAny() to return %v, got %v", want, o)
}

func TestJQReadFromAccessorNotAString(t *testing.T) {
	t.Parallel()

	s, err := util.JQReadFrom[string](context.Background(), ".simple", map[string]any{
		"simple": 1,
	})

	assert.Error(t, err, "Expected JQReadFrom() to return an error")
	assert.Equal(t, "", s, "Expected JQReadFrom() to return an empty string")
}

func TestJQReadFromAccessorBadAccessor(t *testing.T) {
	t.Parallel()

	var s string
	var err error

	s, err = util.JQReadFrom[string](context.Background(), ".simple", map[string]any{
		"not_so_simple": 1,
	})

	assert.True(t, errors.Is(err, util.ErrNoValueFound), "Expected JQReadFrom() to return ErrNoValueFound")
	assert.Equal(t, "", s, "Expected JQReadFrom() to return an empty string")
}

func TestJQReadFromAccessorBadAny(t *testing.T) {
	t.Parallel()

	var a any
	var err error

	a, err = util.JQReadFrom[any](context.Background(), ".simple", map[string]any{
		"not_so_simple": 1,
	})

	assert.True(t, errors.Is(err, util.ErrNoValueFound), "Expected JQReadFrom() to return ErrNoValueFound")
	assert.Nil(t, a, "Expected JQReadFrom() to return nil")
}

func TestJQReadFromAccessorInvalidObject(t *testing.T) {
	t.Parallel()

	a, err := util.JQReadFrom[any](context.Background(), ".simple", "invalid")

	assert.Error(t, err, "Expected JQReadFrom() to return an error")
	assert.Nil(t, a, "Expected JQReadFrom() to return nil")
}

func TestJQReadFromAccessorNoMatch(t *testing.T) {
	t.Parallel()

	o, err := util.JQReadFrom[any](context.Background(), ".you.shall.not.match", map[string]any{
		"nested": map[string]any{
			"object": map[string]any{
				"one":   1,
				"two":   2,
				"three": 3,
			},
		},
	})

	assert.True(t, errors.Is(err, util.ErrNoValueFound), "Expected JQReadFrom() to return ErrNoValueFound")
	assert.Nil(t, o, "Expected jQReadAsAny() to return nil, got %v", o)
}

func TestJQReadFromAccessorInvalid(t *testing.T) {
	t.Parallel()

	o, err := util.JQReadFrom[map[string]any](context.Background(), ".object.one[0]", map[string]any{
		"object": map[string]any{
			"one":   1,
			"two":   2,
			"three": 3,
		},
	})

	assert.Error(t, err, "Expected JQReadFrom() to return an error")
	assert.Nil(t, o, "Expected JQReadFrom() to return nil")
}
