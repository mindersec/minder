// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package jq_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/jq"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
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
						Constant: structpb.NewStringValue("b"),
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
					Constant: structpb.NewStringValue("simple"),
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
					Constant: structpb.NewNumberValue(1),
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
					Constant: structpb.NewBoolValue(false),
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
					Constant: structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStringValue("a"),
							structpb.NewStringValue("b"),
							structpb.NewStringValue("c"),
						},
					}),
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

			_, err = jqe.Eval(context.Background(), tt.args.pol, nil, &interfaces.Ingested{Object: tt.args.obj})
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
					Constant: structpb.NewBoolValue(true),
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
					Constant: structpb.NewBoolValue(false),
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
					Constant: structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStringValue("a"),
						},
					}),
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

			_, err = jqe.Eval(context.Background(), tt.args.pol, nil, &interfaces.Ingested{Object: tt.args.obj})
			assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "Got unexpected error")
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

			_, err := jq.NewJQEvaluator(tt.assertions)
			assert.Error(t, err, "Expected error")
		})
	}
}

func TestEvaluationDetailRendering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msg     string
		msgArgs []any
		tmpl    string
		args    any
		error   string
		details string
	}{
		// JQ template
		{
			name: "JQ template with different actual and expected values",
			msg:  "this is the message",
			tmpl: templates.JqTemplate,
			args: map[string]any{
				"path":     ".simple",
				"expected": true,
				"actual":   false,
			},
			error: "evaluation failure: this is the message",
			details: "The detected configuration does not match the desired configuration:\n" +
				"Expected \".simple\" to equal true, but was false.",
		},
		{
			name: "JQ template with different actual value equal nil",
			msg:  "this is the message",
			tmpl: templates.JqTemplate,
			args: map[string]any{
				"path":     ".simple",
				"expected": 5,
				"actual":   nil,
			},
			error: "evaluation failure: this is the message",
			details: "The detected configuration does not match the desired configuration:\n" +
				"Expected \".simple\" to equal 5, but was not set.",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := evalerrors.NewDetailedErrEvaluationFailed(
				tt.tmpl,
				tt.args,
				tt.msg,
				tt.msgArgs...,
			)

			require.Equal(t, tt.error, err.Error())
			evalErr, ok := err.(*evalerrors.EvaluationError)
			require.True(t, ok)
			require.Equal(t, tt.details, evalErr.Details())
		})
	}
}
