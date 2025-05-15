// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
		{
			name:    "uuid name",
			args:    args{name: "123e4567-e89b-12d3-a456-426614174000"},
			wantErr: true,
		},
		{
			name:    "uuid-like name",
			args:    args{name: "l23e4567-e89b-12d3-a4S6-426614174O00"},
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
