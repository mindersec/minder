// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package algorithms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Type
		wantErr bool
	}{
		{
			name:    "valid aes-256-cfb",
			input:   "aes-256-cfb",
			want:    Aes256Cfb,
			wantErr: false,
		},
		{
			name:    "valid aes-256-gcm",
			input:   "aes-256-gcm",
			want:    Aes256Gcm,
			wantErr: false,
		},
		{
			name:    "invalid algorithm",
			input:   "des",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := TypeFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrUnknownAlgorithm)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewFromType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		algoType Type
		wantErr  bool
	}{
		{
			name:     "valid aes-256-cfb",
			algoType: Aes256Cfb,
			wantErr:  false,
		},
		{
			name:     "valid aes-256-gcm",
			algoType: Aes256Gcm,
			wantErr:  false,
		},
		{
			name:     "invalid algorithm",
			algoType: Type("des"),
			wantErr:  true,
		},
		{
			name:     "empty algorithm",
			algoType: Type(""),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewFromType(tt.algoType)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrUnknownAlgorithm)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}
