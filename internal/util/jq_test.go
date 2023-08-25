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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stacklok/mediator/internal/util"
)

func TestJQGetValuesFromAccessorValid(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx  context.Context
		path string
		obj  any
	}
	tests := []struct {
		name string
		args args
		want any
	}{
		{
			name: "string",
			args: args{
				ctx:  context.Background(),
				path: ".simple",
				obj: map[string]any{
					"simple": "value",
				},
			},
			want: "value",
		},
		{
			name: "number",
			args: args{
				ctx:  context.Background(),
				path: ".number",
				obj: map[string]any{
					"number": 1,
				},
			},
			want: 1,
		},
		{
			name: "boolean",
			args: args{
				ctx:  context.Background(),
				path: ".boolean",
				obj: map[string]any{
					"boolean": true,
				},
			},
			want: true,
		},
		{
			name: "array",
			args: args{
				ctx:  context.Background(),
				path: ".array",
				obj: map[string]any{
					"array": []any{
						"one",
						"two",
						"three",
					},
				},
			},
			want: []any{
				"one",
				"two",
				"three",
			},
		},
		{
			name: "object",
			args: args{
				ctx:  context.Background(),
				path: ".object",
				obj: map[string]any{
					"object": map[string]any{
						"one":   1,
						"two":   2,
						"three": 3,
					},
				},
			},
			want: map[string]any{
				"one":   1,
				"two":   2,
				"three": 3,
			},
		},
		{
			name: "nested",
			args: args{
				ctx:  context.Background(),
				path: ".nested.object",
				obj: map[string]any{
					"nested": map[string]any{
						"object": map[string]any{
							"one":   1,
							"two":   2,
							"three": 3,
						},
					},
				},
			},
			want: map[string]any{
				"one":   1,
				"two":   2,
				"three": 3,
			},
		},
		{
			name: "nested array",
			args: args{
				ctx:  context.Background(),
				path: ".nested.array",
				obj: map[string]any{
					"nested": map[string]any{
						"array": []any{
							"one",
							"two",
							"three",
						},
					},
				},
			},
			want: []any{
				"one",
				"two",
				"three",
			},
		},
		{
			// This shouldn't fail, but it should return nil
			name: "invalid path",
			args: args{
				ctx:  context.Background(),
				path: ".invalid",
				obj: map[string]any{
					"simple": "value",
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := util.JQGetValuesFromAccessor(tt.args.ctx, tt.args.path, tt.args.obj)
			assert.NoError(t, err, "Unexpected error processing JQGetValuesFromAccessor()")
			assert.True(t, reflect.DeepEqual(got, tt.want), "Expected JQGetValuesFromAccessor() to return %v, got %v", tt.want, got)
		})
	}
}

func TestJQGetValuesFromAccessorInvalid(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx  context.Context
		path string
		obj  any
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "invalid object",
			args: args{
				ctx:  context.Background(),
				path: ".simple",
				obj:  "invalid",
			},
		},
		{
			name: "invalid path",
			args: args{
				ctx:  context.Background(),
				path: ".simple.invalid[0]",
				obj: map[string]any{
					"simple": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := util.JQGetValuesFromAccessor(tt.args.ctx, tt.args.path, tt.args.obj)
			assert.Nil(t, got, "Expected JQGetValuesFromAccessor() to return nil, got %v", got)
			t.Log(err)
			assert.Error(t, err, "JQGetValuesFromAccessor() should have returned an error")
		})
	}
}
