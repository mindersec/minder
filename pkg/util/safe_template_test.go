// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package util provides helper functions for minder
package util_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/pkg/util"
)

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

			tmpl, err := util.NewSafeTextTemplate(tt.args.tmpl, tt.args.name)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.IsType(t, &util.SafeTemplate{}, tmpl, "expected *util.SafeTemplate")
			}
		})
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl, err := util.NewSafeHTMLTemplate(tt.args.tmpl, tt.args.name)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.IsType(t, &util.SafeTemplate{}, tmpl, "expected *util.SafeTemplate")
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
		{
			name: "too large apiBaseURL",
			args: args{
				method:     http.MethodGet,
				apiBaseURL: verybigstring(util.CurlCmdMaxSize + 1),
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

			cmd, err := util.GenerateCurlCommand(
				context.TODO(), tt.args.method, tt.args.apiBaseURL, tt.args.endpoint, tt.args.body)
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

func verybigstring(n int) string {
	s := "a"
	for i := 0; i < n; i++ {
		s += "a"
	}
	return s
}

func TestRenderStructPB(t *testing.T) {
	t.Parallel()

	const limit = 1024

	type args struct {
		tmpl string
		s    any
	}
	tests := []struct {
		name     string
		args     args
		expected string
		wantErr  bool
	}{
		{
			name: "asMap: valid template",
			args: args{
				tmpl: "{{ with $m := asMap . }}{{ $m.name }}{{ end }}",
				s: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name": {
							Kind: &structpb.Value_StringValue{
								StringValue: "test",
							},
						},
					},
				},
			},
			expected: "test",
			wantErr:  false,
		},
		{
			name: "asMap: using wrong key",
			args: args{
				tmpl: "{{ with $m := asMap . }}{{ $m.name2 }}{{ end }}",
				s: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name": {
							Kind: &structpb.Value_StringValue{
								StringValue: "test",
							},
						},
					},
				},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "asMap: using wrong type",
			args: args{
				tmpl: "{{ with $m := asMap . }}{{ $m.name }}{{ end }}",
				s:    "test",
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "asMap: nil structpb",
			args: args{
				tmpl: "{{ with $m := asMap . }}{{ $m.name }}{{ end }}",
				s:    nil,
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "mapGet: valid with map[string]any",
			args: args{
				tmpl: "{{ mapGet . \"name\" }}",
				s: map[string]any{
					"name": "test",
				},
			},
			expected: "test",
			wantErr:  false,
		},
		{
			name: "mapGet: valid with asMapper",
			args: args{
				tmpl: "{{ mapGet . \"name\" }}",
				s: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name": {
							Kind: &structpb.Value_StringValue{
								StringValue: "test",
							},
						},
					},
				},
			},
			expected: "test",
			wantErr:  false,
		},
		{
			name: "mapGet: using wrong key",
			args: args{
				tmpl: "{{ mapGet . \"name2\" }}",
				s: map[string]any{
					"name": "test",
				},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "mapGet: using wrong type",
			args: args{
				tmpl: "{{ mapGet . \"name\" }}",
				s:    "test",
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "mapGet: nil map",
			args: args{
				tmpl: "{{ mapGet . \"name\" }}",
				s:    nil,
			},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl, err := util.NewSafeTextTemplate(&tt.args.tmpl, "test")
			// We're not testing the template parsing here
			require.NoError(t, err, "unexpected error")

			out, err := tmpl.Render(context.Background(), tt.args.s, limit)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.Equal(t, tt.expected, out, "expected output")
			}
		})
	}

}
