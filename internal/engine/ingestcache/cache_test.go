// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package ingestcache_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/mediator/internal/engine/ingestcache"
	"github.com/stacklok/mediator/internal/engine/ingester/artifact"
	"github.com/stacklok/mediator/internal/engine/ingester/builtin"
	"github.com/stacklok/mediator/internal/engine/ingester/diff"
	"github.com/stacklok/mediator/internal/engine/ingester/git"
	"github.com/stacklok/mediator/internal/engine/ingester/rest"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	mediatorv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func TestCache(t *testing.T) {
	t.Parallel()

	type args struct {
		in0 engif.Ingester
		in1 protoreflect.ProtoMessage
		in2 *engif.EvalStatusParams
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "REST",
			args: args{
				in0: &rest.Ingestor{},
				in1: &mediatorv1.RestType{
					Endpoint: "http://localhost:8080",
				},
				in2: &engif.EvalStatusParams{
					Rule: &mediatorv1.Profile_Rule{
						Params: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"foo": {
									Kind: &structpb.Value_StringValue{
										StringValue: "bar",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "REST with no params",
			args: args{
				in0: &rest.Ingestor{},
				in1: &mediatorv1.RestType{
					Endpoint: "http://localhost:8080",
				},
				in2: &engif.EvalStatusParams{
					Rule: &mediatorv1.Profile_Rule{},
				},
			},
		},
		{
			name: "Builtin",
			args: args{
				in0: &builtin.BuiltinRuleDataIngest{},
				in1: &mediatorv1.BuiltinType{
					Method: "foo",
				},
				in2: &engif.EvalStatusParams{
					Rule: &mediatorv1.Profile_Rule{
						Params: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"bar": {
									Kind: &structpb.Value_StringValue{
										StringValue: "barbar",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Artifact",
			args: args{
				in0: &artifact.Ingest{},
				in1: nil, // Artifacts have no config
				in2: &engif.EvalStatusParams{
					Rule: &mediatorv1.Profile_Rule{
						Params: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"baz": {
									Kind: &structpb.Value_StringValue{
										StringValue: "bazbaz",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Diff",
			args: args{
				in0: &diff.Diff{},
				in1: &mediatorv1.DiffType{
					Ecosystems: []*mediatorv1.DiffType_Ecosystem{
						{
							Name:    "beer",
							Depfile: "is.good",
						},
					},
				},
				in2: &engif.EvalStatusParams{
					Rule: &mediatorv1.Profile_Rule{
						Params: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"qux": {
									Kind: &structpb.Value_StringValue{
										StringValue: "quxqux",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Git",
			args: args{
				in0: &git.Git{},
				in1: &mediatorv1.GitType{
					CloneUrl: "http://localhost:8080",
				},
				in2: &engif.EvalStatusParams{
					Rule: &mediatorv1.Profile_Rule{
						Params: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"quux": {
									Kind: &structpb.Value_StringValue{
										StringValue: "quxqux",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run("cache test "+tt.name, func(t *testing.T) {
			t.Parallel()
			cache := ingestcache.NewCache()

			_, ok := cache.Get(tt.args.in0, tt.args.in1, tt.args.in2)
			require.False(t, ok, "cache should be empty")

			res := &engif.Result{
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
		tt := tt

		t.Run("noopcache test "+tt.name, func(t *testing.T) {
			t.Parallel()
			cache := ingestcache.NewNoopCache()

			_, ok := cache.Get(tt.args.in0, tt.args.in1, tt.args.in2)
			require.False(t, ok, "cache should be empty")

			res := &engif.Result{
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
