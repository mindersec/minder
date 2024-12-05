// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"strconv"
	"testing"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util"
)

func TestGetConfigValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		key          string
		flagName     string
		defaultValue interface{}
		flagValue    interface{}
		flagSet      bool
		expected     interface{}
	}{
		{
			name:         "string flag set",
			key:          "testString",
			flagName:     "test-string",
			defaultValue: "default",
			flagValue:    "newValue",
			flagSet:      true,
			expected:     "newValue",
		},
		{
			name:         "int flag set",
			key:          "testInt",
			flagName:     "test-int",
			defaultValue: 1,
			flagValue:    42,
			flagSet:      true,
			expected:     42,
		},
		{
			name:         "flag not set",
			key:          "testFlagNotSet",
			flagName:     "test-notset",
			defaultValue: "default",
			flagValue:    "",
			flagSet:      false,
			expected:     "default",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := viper.New()

			v.SetDefault(tc.key, tc.defaultValue)

			cmd := &cobra.Command{}
			switch tc.defaultValue.(type) {
			case string:
				cmd.Flags().String(tc.flagName, tc.defaultValue.(string), "")
			case int:
				cmd.Flags().Int(tc.flagName, tc.defaultValue.(int), "")
			}
			// bind the flag to viper
			err := v.BindPFlag(tc.key, cmd.Flags().Lookup(tc.flagName))
			if err != nil {
				t.Fatalf("Error binding flag %s: %v", tc.flagName, err)
			}
			if tc.flagSet {
				switch tc.flagValue.(type) {
				case string:
					err := cmd.Flags().Set(tc.flagName, tc.flagValue.(string))
					if err != nil {
						t.Fatalf("Error setting flag %s: %v", tc.flagName, err)
					}
				case int:
					err := cmd.Flags().Set(tc.flagName, strconv.Itoa(tc.flagValue.(int)))
					if err != nil {
						t.Fatalf("Error setting flag %s: %v", tc.flagName, err)
					}
				}
			}

			result := v.Get(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInt32FromString(t *testing.T) {
	t.Parallel()

	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    int32
		wantErr bool
	}{
		{
			name:    "valid int32",
			args:    args{v: "42"},
			want:    42,
			wantErr: false,
		},
		{
			name:    "valid int32 negative",
			args:    args{v: "-42"},
			want:    -42,
			wantErr: false,
		},
		{
			name:    "big int32",
			args:    args{v: "2147483647"},
			want:    2147483647,
			wantErr: false,
		},
		{
			name:    "big int32 negative",
			args:    args{v: "-2147483648"},
			want:    -2147483648,
			wantErr: false,
		},
		{
			name:    "too big int32",
			args:    args{v: "12147483648"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "valid zero",
			args:    args{v: "0"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "invalid int32",
			args:    args{v: "invalid"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			args:    args{v: ""},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := util.Int32FromString(tt.args.v)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Zero(t, got, "expected zero")
				return
			}

			assert.Equal(t, tt.want, got, "result didn't match")
		})
	}
}
