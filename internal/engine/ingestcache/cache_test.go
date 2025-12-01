// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ingestcache_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/engine/ingestcache"
	"github.com/mindersec/minder/internal/engine/ingester/artifact"
	"github.com/mindersec/minder/internal/engine/ingester/builtin"
	"github.com/mindersec/minder/internal/engine/ingester/diff"
	"github.com/mindersec/minder/internal/engine/ingester/git"
	"github.com/mindersec/minder/internal/engine/ingester/rest"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func TestCache(t *testing.T) {
	t.Parallel()

	type args struct {
		in0 interfaces.Ingester
		in1 protoreflect.ProtoMessage
		in2 map[string]any
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "REST",
			args: args{
				in0: &rest.Ingestor{},
				in1: &minderv1.RestType{
					Endpoint: "http://localhost:8080",
				},
				in2: map[string]any{
					"foo": "bar",
				},
			},
		},
		{
			name: "REST with no params",
			args: args{
				in0: &rest.Ingestor{},
				in1: &minderv1.RestType{
					Endpoint: "http://localhost:8080",
				},
				in2: nil,
			},
		},
		{
			name: "Builtin",
			args: args{
				in0: &builtin.RuleDataIngest{},
				in1: &minderv1.BuiltinType{
					Method: "foo",
				},
				in2: map[string]any{
					"bar": "barbar",
				},
			},
		},
		{
			name: "Artifact",
			args: args{
				in0: &artifact.Ingest{},
				in1: nil, // Artifacts have no config
				in2: map[string]any{
					"baz": "bazbaz",
				},
			},
		},
		{
			name: "Diff",
			args: args{
				in0: &diff.Diff{},
				in1: &minderv1.DiffType{
					Ecosystems: []*minderv1.DiffType_Ecosystem{
						{
							Name:    "beer",
							Depfile: "is.good",
						},
					},
				},
				in2: map[string]any{
					"qux": "quxqux",
				},
			},
		},
		{
			name: "Git",
			args: args{
				in0: &git.Git{},
				in1: &minderv1.GitType{
					CloneUrl: "http://localhost:8080",
				},
				in2: map[string]any{
					"quux": "quxqux",
				},
			},
		},
	}
	for _, tt := range tests {

		t.Run("cache test "+tt.name, func(t *testing.T) {
			t.Parallel()
			cache := ingestcache.NewCache()

			_, ok := cache.Get(tt.args.in0, tt.args.in1, tt.args.in2)
			require.False(t, ok, "cache should be empty")

			res := &interfaces.Ingested{
				Object: map[string]any{
					"foo": "bar",
				},
			}

			cache.Set(tt.args.in0, tt.args.in1, tt.args.in2, res)

			res2, ok := cache.Get(tt.args.in0, tt.args.in1, tt.args.in2)
			require.True(t, ok, "cache should have value")
			require.Equal(t, res, res2)
		})
	}

	for _, tt := range tests {

		t.Run("noopcache test "+tt.name, func(t *testing.T) {
			t.Parallel()
			cache := ingestcache.NewNoopCache()

			_, ok := cache.Get(tt.args.in0, tt.args.in1, tt.args.in2)
			require.False(t, ok, "cache should be empty")

			res := &interfaces.Ingested{
				Object: map[string]any{
					"foo": "bar",
				},
			}

			cache.Set(tt.args.in0, tt.args.in1, tt.args.in2, res)

			_, ok = cache.Get(tt.args.in0, tt.args.in1, tt.args.in2)
			require.False(t, ok, "cache should still  be empty")
		})
	}
}
