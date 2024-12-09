// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package structured

import (
	"testing"

	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewStructDataSource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		sds     *minderv1.StructDataSource
		mustErr bool
	}{
		{"nil-def", nil, true},
		{"no-def", &minderv1.StructDataSource{}, true},
		{"invalid-def", &minderv1.StructDataSource{
			Def: map[string]*minderv1.StructDataSource_Def{"test": nil},
		}, true},
		{"success", &minderv1.StructDataSource{
			Def: map[string]*minderv1.StructDataSource_Def{
				"test": {
					Path: &minderv1.StructDataSource_Def_Path{FileName: "test.yaml"},
				},
			},
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewStructDataSource(tc.sds)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
