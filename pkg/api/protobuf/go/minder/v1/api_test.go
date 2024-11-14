// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResourceProto(t *testing.T) {
	t.Parallel()

	type args struct {
		r  io.Reader
		rm *DataSource
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test data source parsing",
			args: args{
				r: strings.NewReader(`
version: v1
type: "data-source"
name: "foo"
rest:
  def:
    foo:
      input_schema:
        properties:
          foo:
            type: "string"
            description: "foo"
        required:
          - foo
      endpoint: "http://example.com"
      method: "GET"
      headers:
        Content-Type: "application/json"
`),
				rm: &DataSource{},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.wantErr(t, ParseResourceProto(tt.args.r, tt.args.rm), fmt.Sprintf("ParseResourceProto(%v, %v)", tt.args.r, tt.args.rm))
			t.Logf("Resource: %+v", tt.args.rm)
		})
	}
}
