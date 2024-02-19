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
	htmltemplate "html/template"
	"net/http"
	"testing"
	"text/template"

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

func TestParseNewTemplate(t *testing.T) {
	t.Parallel()

	type args struct {
		tmpl *string
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid template",
			args: args{
				tmpl: stringPtr("{{ .Name }}"),
				name: "test",
			},
			wantErr: false,
		},
		{
			name: "empty template",
			args: args{
				tmpl: stringPtr(""),
				name: "test",
			},
			wantErr: true,
		},
		{
			name: "nil template",
			args: args{
				tmpl: nil,
				name: "test",
			},
			wantErr: true,
		},
		{
			name: "malformed template",
			args: args{
				tmpl: stringPtr("{{ .Name"),
				name: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl, err := util.ParseNewTextTemplate(tt.args.tmpl, tt.args.name)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.IsType(t, &template.Template{}, tmpl, "expected *template.Template")
			}
		})
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl, err := util.ParseNewHtmlTemplate(tt.args.tmpl, tt.args.name)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.IsType(t, &htmltemplate.Template{}, tmpl, "expected *htmltemplate.Template")
			}
		})
	}
}

func TestGenerateCurlCommand(t *testing.T) {
	t.Parallel()

	type args struct {
		method     string
		apiBaseURL string
		endpoint   string
		body       string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid template",
			args: args{
				method:     http.MethodGet,
				apiBaseURL: "https://api.stacklok.com",
				endpoint:   "/v1/projects",
				body:       "",
			},
			wantErr: false,
		},
		{
			name: "valid template with body",
			args: args{
				method:     http.MethodPost,
				apiBaseURL: "https://api.stacklok.com",
				endpoint:   "/v1/projects",
				body:       "test",
			},
			wantErr: false,
		},
		{
			name: "empty method",
			args: args{
				method:     "",
				apiBaseURL: "https://api.stacklok.com",
				endpoint:   "/v1/projects",
				body:       "",
			},
			wantErr: true,
		},
		{
			name: "empty apiBaseURL",
			args: args{
				method:     http.MethodGet,
				apiBaseURL: "",
				endpoint:   "/v1/projects",
				body:       "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd, err := util.GenerateCurlCommand(tt.args.method, tt.args.apiBaseURL, tt.args.endpoint, tt.args.body)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.NotEmpty(t, cmd, "expected command")

				assert.Containsf(t, cmd, tt.args.method, "expected method %s in command %s", tt.args.method, cmd)
				assert.Contains(t, cmd, tt.args.apiBaseURL, "expected apiBaseURL in command")
				assert.Contains(t, cmd, tt.args.endpoint, "expected endpoint in command")
				if len(tt.args.body) > 0 {
					assert.Contains(t, cmd, tt.args.body, "expected body in command")
				}
			}
		})
	}

}

func stringPtr(s string) *string {
	return &s
}
