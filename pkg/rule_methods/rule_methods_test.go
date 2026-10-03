// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rule_methods

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRuleMethods_GetMethod(t *testing.T) {
	t.Parallel()

	rm := &RuleMethods{}

	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{
			name:    "existing method",
			method:  "Passthrough",
			wantErr: false,
		},
		{
			name:    "non-existing method",
			method:  "DoesNotExist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := rm.GetMethod(tt.method)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, reflect.Value{}, got)
			} else {
				require.NoError(t, err)
				require.True(t, got.IsValid())
			}
		})
	}
}

func TestRuleMethods_Passthrough(t *testing.T) {
	t.Parallel()

	rm := &RuleMethods{}
	ctx := context.Background()

	// Use a simple known protobuf message
	msg, err := structpb.NewStruct(map[string]interface{}{
		"key": "value",
	})
	require.NoError(t, err)

	out, err := rm.Passthrough(ctx, msg)
	require.NoError(t, err)

	// Validate output
	require.Contains(t, string(out), `"key":"value"`)
}
