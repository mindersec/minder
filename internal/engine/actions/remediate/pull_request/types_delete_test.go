// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestDeleteModification_FileIsRemoved(t *testing.T) {
	t.Parallel()

	fs := memfs.New()
	require.NoError(t, fs.MkdirAll(".github/workflows", 0755))
	require.NoError(t, billyutil.WriteFile(fs, ".github/workflows/build.yml", []byte("content"), 0644))

	mod, err := newDeleteModification(&modificationConstructorParams{
		bfs: fs,
		prCfg: &pb.RuleType_Definition_Remediate_PullRequestRemediation{
			Method: minderDeleteModification,
			Params: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"patterns": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"pattern": structpb.NewStringValue(".github/workflows/*.yml"),
									"type":    structpb.NewStringValue("glob"),
								},
							}),
						},
					}),
				},
			},
		},
	})
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

	mod, err := newDeleteModification(&modificationConstructorParams{
		bfs: fs,
		prCfg: &pb.RuleType_Definition_Remediate_PullRequestRemediation{
			Method: minderDeleteModification,
			Params: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"patterns": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"pattern": structpb.NewStringValue(".github/workflows/*.yml"),
									"type":    structpb.NewStringValue("glob"),
								},
							}),
						},
					}),
				},
			},
		},
	})
	require.NoError(t, err)

	require.NoError(t, mod.createFsModEntries(t.Context(), nil, nil))

	entries, err := mod.modifyFs()
	require.NoError(t, err)
	require.Len(t, entries, 0)
}
