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

// Package projects contains utilities for working with projects.
package projects

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid name",
			args:    args{name: "valid-name"},
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			args:    args{name: "valid-name-123"},
			wantErr: false,
		},
		{
			name:    "valid DNS name",
			args:    args{name: "valid-name-123.stacklok.com"},
			wantErr: false,
		},
		{
			name:    "invalid name",
			args:    args{name: "invalid name"},
			wantErr: true,
		},
		{
			name:    "empty name",
			args:    args{name: ""},
			wantErr: true,
		},
		{
			name: "name too long",
			// 65 characters
			args:    args{name: strings.Repeat("a", 65)},
			wantErr: true,
		},
		{
			name:    "slash in the name",
			args:    args{name: "name/with/slash"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateName(tt.args.name)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
