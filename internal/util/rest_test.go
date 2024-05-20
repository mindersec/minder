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

// Package util provides helper functions for minder
package util_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stacklok/minder/internal/util"
)

func TestHttpMethodFromString(t *testing.T) {
	t.Parallel()

	type args struct {
		inMeth string
		dfl    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "valid method",
			args: args{
				inMeth: "GET",
				dfl:    http.MethodGet,
			},
			want: http.MethodGet,
		},
		{
			name: "lowercase method",
			args: args{
				inMeth: "get",
				dfl:    http.MethodGet,
			},
			want: http.MethodGet,
		},
		{
			name: "empty method",
			args: args{
				inMeth: "",
				dfl:    http.MethodPost,
			},
			want: http.MethodPost,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := util.HttpMethodFromString(tt.args.inMeth, tt.args.dfl)
			assert.Equal(t, tt.want, got, "expected %s, got %s", tt.want, got)
		})
	}
}
