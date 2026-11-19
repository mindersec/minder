// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func deletePattern(pattern, typ string) *structpb.Value {
	return structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"pattern": structpb.NewStringValue(pattern),
			"type":    structpb.NewStringValue(typ),
		},
	})
}

func deleteParams(patterns ...*structpb.Value) *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"patterns": structpb.NewListValue(&structpb.ListValue{Values: patterns}),
		},
	}
}

func newDeleteModificationForTest(
	fs billy.Filesystem,
	params *structpb.Struct,
) (fsModifier, error) {
	return newDeleteModification(&modificationConstructorParams{
		bfs: fs,
		prCfg: &pb.RuleType_Definition_Remediate_PullRequestRemediation{
			Method: minderDeleteModification,
			Params: params,
		},
	})
}

func TestDeleteModification_FileIsRemoved(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	require.NoError(t, fs.MkdirAll(".github/workflows", 0755))
	require.NoError(t, billyutil.WriteFile(fs, ".github/workflows/build.yml", []byte("content"), 0644))

	mod, err := newDeleteModificationForTest(fs, deleteParams(deletePattern(".github/workflows/*.yml", "glob")))
	require.NoError(t, err)

	require.NoError(t, mod.createFsModEntries(t.Context(), nil, nil))

	entries, err := mod.modifyFs()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.True(t, entries[0].Delete)

	_, err = fs.Stat(".github/workflows/build.yml")
	require.Error(t, err, "file should have been deleted")
}

func TestDeleteModification_NoMatchIsOk(t *testing.T) {
	t.Parallel()

	fs := memfs.New()

	mod, err := newDeleteModificationForTest(fs, deleteParams(deletePattern(".github/workflows/*.yml", "glob")))
	require.NoError(t, err)

	require.NoError(t, mod.createFsModEntries(t.Context(), nil, nil))

	entries, err := mod.modifyFs()
	require.NoError(t, err)
	require.Len(t, entries, 0)
}

func TestDeleteModification_SkipsDirectories(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	require.NoError(t, fs.MkdirAll(".github/workflows/nested", 0755))
	require.NoError(t, billyutil.WriteFile(fs, ".github/workflows/build.yml", []byte("content"), 0644))
	require.NoError(t, billyutil.WriteFile(fs, ".github/workflows/nested/other.yml", []byte("content"), 0644))

	mod, err := newDeleteModificationForTest(fs, deleteParams(deletePattern(".github/workflows/*", "glob")))
	require.NoError(t, err)

	require.NoError(t, mod.createFsModEntries(t.Context(), nil, nil))

	entries, err := mod.modifyFs()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, ".github/workflows/build.yml", entries[0].Path)
	require.True(t, entries[0].Delete)

	_, err = fs.Stat(".github/workflows/build.yml")
	require.Error(t, err, "file should have been deleted")

	info, err := fs.Stat(".github/workflows/nested")
	require.NoError(t, err, "directory should have been kept")
	require.True(t, info.IsDir())
}

func TestDeleteModification_ConstructorErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  *structpb.Struct
		wantErr string
	}{
		{
			name:    "missing params",
			params:  nil,
			wantErr: "delete modification requires params",
		},
		{
			name:    "missing patterns",
			params:  &structpb.Struct{},
			wantErr: "delete modification requires patterns",
		},
		{
			name: "patterns is not a list",
			params: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"patterns": structpb.NewStringValue(".github/workflows/*.yml"),
				},
			},
			wantErr: "patterns must be a list",
		},
		{
			name:    "non-object entry",
			params:  deleteParams(structpb.NewStringValue(".github/workflows/*.yml")),
			wantErr: "patterns[0] must be an object",
		},
		{
			name:    "missing pattern",
			params:  deleteParams(deletePattern("", "glob")),
			wantErr: "patterns[0] missing required field: pattern",
		},
		{
			name:    "missing type",
			params:  deleteParams(deletePattern(".github/workflows/*.yml", "")),
			wantErr: "patterns[0] missing required field: type",
		},
		{
			name:    "unsupported type",
			params:  deleteParams(deletePattern(".github/workflows/*.yml", "regex")),
			wantErr: `patterns[0] unsupported type "regex" (only "glob" is supported)`,
		},
		{
			name:    "empty list",
			params:  deleteParams(),
			wantErr: "delete modification requires at least one valid pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := newDeleteModificationForTest(memfs.New(), tt.params)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
