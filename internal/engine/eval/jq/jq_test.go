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

package jq_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/jq"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewJQEvaluatorValid(t *testing.T) {
	t.Parallel()

	type args struct {
		assertions []*pb.RuleType_Definition_Eval_JQComparison
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "valid single rule",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{
					{
						Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
						Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
					},
				},
			},
		},
		{
			name: "valid multiple rules",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{
					{
						Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".a",
						},
						Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".a",
						},
					},
					{
						Constant: &structpb.Value{
							Kind: &structpb.Value_StringValue{
								StringValue: "b",
							},
						},
						Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".b",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := jq.NewJQEvaluator(tt.args.assertions)
			assert.NoError(t, err, "Got unexpected error")
			assert.NotNil(t, got, "Got unexpected nil")
		})
	}
}

func TestNewJQEvaluatorInvalid(t *testing.T) {
	t.Parallel()

	type args struct {
		assertions []*pb.RuleType_Definition_Eval_JQComparison
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "invalid nil assertions",
			args: args{
				assertions: nil,
			},
		},
		{
			name: "invalid empty assertions",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{},
			},
		},
		{
			name: "invalid nil profile and constant accessor",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{
					{
						Profile:  nil,
						Constant: nil,
						Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
					},
				},
			},
		},
		{
			name: "invalid empty profile accessor",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{
					{
						Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{},
						Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
					},
				},
			},
		},
		{
			name: "invalid nil ingested accessor",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{
					{
						Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
						Ingested: nil,
					},
				},
			},
		},
		{
			name: "invalid empty ingested accessor",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{
					{
						Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
						Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{},
					},
				},
			},
		},
		{
			name: "one valid accessor and one invalid",
			args: args{
				assertions: []*pb.RuleType_Definition_Eval_JQComparison{
					{
						Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
						Ingested: nil,
					},
					{
						Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
						Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
							Def: ".",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := jq.NewJQEvaluator(tt.args.assertions)
			assert.Error(t, err, "expected error")
			assert.Nil(t, got, "expected nil")
		})
	}
}

func TestValidJQEvals(t *testing.T) {
	t.Parallel()

	type args struct {
		pol map[string]any
		obj any
	}
	tests := []struct {
		name       string
		assertions []*pb.RuleType_Definition_Eval_JQComparison
		args       args
	}{
		{
			name: "valid single rule evaluates string",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": "simple",
				},
				obj: map[string]any{
					"simple": "simple",
				},
			},
		},
		{
			name: "valid single rule evaluates constant string",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Constant: &structpb.Value{
						Kind: &structpb.Value_StringValue{
							StringValue: "simple",
						},
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{},
				obj: map[string]any{
					"simple": "simple",
				},
			},
		},
		{
			name: "valid single rule evaluates int",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": 1,
				},
				obj: map[string]any{
					"simple": 1,
				},
			},
		},
		{
			name: "valid single rule evaluates constant int",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Constant: &structpb.Value{
						Kind: &structpb.Value_NumberValue{
							NumberValue: 1,
						},
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{},
				obj: map[string]any{
					"simple": 1,
				},
			},
		},
		{
			name: "valid single rule evaluates bool",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": false,
				},
				obj: map[string]any{
					"simple": false,
				},
			},
		},
		{
			name: "valid single rule evaluates constant bool",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Constant: &structpb.Value{
						Kind: &structpb.Value_BoolValue{
							BoolValue: false,
						},
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{},
				obj: map[string]any{
					"simple": false,
				},
			},
		},
		{
			name: "valid single rule evaluates array",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": []any{"a", "b", "c"},
				},
				obj: map[string]any{
					"simple": []any{"a", "b", "c"},
				},
			},
		},
		{
			name: "valid single rule evaluates constant array",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Constant: &structpb.Value{
						Kind: &structpb.Value_ListValue{
							ListValue: &structpb.ListValue{
								Values: []*structpb.Value{
									{
										Kind: &structpb.Value_StringValue{
											StringValue: "a",
										},
									},
									{
										Kind: &structpb.Value_StringValue{
											StringValue: "b",
										},
									},
									{
										Kind: &structpb.Value_StringValue{
											StringValue: "c",
										},
									},
								},
							},
						},
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{},
				obj: map[string]any{
					"simple": []any{"a", "b", "c"},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jqe, err := jq.NewJQEvaluator(tt.assertions)
			assert.NoError(t, err, "Got unexpected error")
			assert.NotNil(t, jqe, "Got unexpected nil")

			err = jqe.Eval(context.Background(), tt.args.pol, &engif.Result{Object: tt.args.obj})
			assert.NoError(t, err, "Got unexpected error")
		})
	}
}

func TestValidJQEvalsFailed(t *testing.T) {
	t.Parallel()

	type args struct {
		pol map[string]any
		obj any
	}
	tests := []struct {
		name       string
		assertions []*pb.RuleType_Definition_Eval_JQComparison
		args       args
	}{
		{
			name: "string doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": "foo",
				},
				obj: map[string]any{
					"simple": "bar",
				},
			},
		},
		{
			name: "int doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": 123,
				},
				obj: map[string]any{
					"simple": 456,
				},
			},
		},
		{
			name: "bool doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": true,
				},
				obj: map[string]any{
					"simple": false,
				},
			},
		},
		{
			name: "type doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": true,
				},
				obj: map[string]any{
					"simple": 123,
				},
			},
		},
		{
			name: "array doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": []any{"a", "b", "c"},
				},
				obj: map[string]any{
					"simple": []any{"a", "b", "d"},
				},
			},
		},
		{
			name: "accessor doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".should_match",
					},
					// This returns nil
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".no_match",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"should_match": "foo",
				},
				obj: map[string]any{
					"shouldnt_match": "foo",
				},
			},
		},
		{
			name: "constant value doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Constant: &structpb.Value{
						Kind: &structpb.Value_BoolValue{
							BoolValue: true,
						},
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{},
				obj: map[string]any{
					"simple": false,
				},
			},
		},
		{
			name: "constant type doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Constant: &structpb.Value{
						Kind: &structpb.Value_BoolValue{
							BoolValue: false,
						},
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{},
				obj: map[string]any{
					"simple": 123,
				},
			},
		},
		{
			name: "constant array doesn't match",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Constant: &structpb.Value{
						Kind: &structpb.Value_ListValue{
							ListValue: &structpb.ListValue{
								Values: []*structpb.Value{
									{
										Kind: &structpb.Value_StringValue{
											StringValue: "a",
										},
									},
								},
							},
						},
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{},
				obj: map[string]any{
					"simple": []any{"a", "b", "d"},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jqe, err := jq.NewJQEvaluator(tt.assertions)
			assert.NoError(t, err, "Got unexpected error")
			assert.NotNil(t, jqe, "Got unexpected nil")

			err = jqe.Eval(context.Background(), tt.args.pol, &engif.Result{Object: tt.args.obj})
			assert.ErrorIs(t, err, evalerrors.ErrEvaluationFailed, "Got unexpected error")
		})
	}
}

func TestInvalidJQEvals(t *testing.T) {
	t.Parallel()

	type args struct {
		pol map[string]any
		obj any
	}
	tests := []struct {
		name       string
		assertions []*pb.RuleType_Definition_Eval_JQComparison
		args       args
	}{
		{
			name: "invalid profile accessor",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: "invalid | foobar",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": "simple",
				},
				obj: map[string]any{
					"simple": "simple",
				},
			},
		},
		{
			name: "invalid ingested accessor",
			assertions: []*pb.RuleType_Definition_Eval_JQComparison{
				{
					Profile: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: ".simple",
					},
					Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
						Def: "invalid | foobar",
					},
				},
			},
			args: args{
				pol: map[string]any{
					"simple": "simple",
				},
				obj: map[string]any{
					"simple": "simple",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			jqe, err := jq.NewJQEvaluator(tt.assertions)
			assert.NoError(t, err, "Got unexpected error")
			assert.NotNil(t, jqe, "Got unexpected nil")

			err = jqe.Eval(context.Background(), tt.args.pol, &engif.Result{Object: tt.args.obj})
			assert.Error(t, err, "Got unexpected error")
			assert.NotErrorIs(t, err, evalerrors.ErrEvaluationFailed, "Got unexpected error")
		})
	}
}
