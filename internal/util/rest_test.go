// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package util provides helper functions for minder
package util_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/internal/util"
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
