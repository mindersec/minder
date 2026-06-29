// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package eval provides necessary interfaces and implementations for evaluating
// rules.
package eval_test

import (
	"context"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/engine/eval"
	"github.com/mindersec/minder/internal/engine/eval/rego"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func TestNewRuleEvaluatorWorks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		eval *pb.RuleType_Definition_Eval
		out  *interfaces.EvaluationResult
	}{
		{
			name: "JQ",
			eval: &pb.RuleType_Definition_Eval{
				Type: "jq",
				Jq: []*pb.RuleType_Definition_Eval_JQComparison{
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
			out: nil,
		},
		{
			name: "Rego defaults",
			eval: &pb.RuleType_Definition_Eval{
				Type: "rego",
				Rego: &pb.RuleType_Definition_Eval_Rego{
					Type: rego.DenyByDefaultEvaluationType.String(),
					Def:  "package minder\n\nimport rego.v1\n\ndefault allow := false\n\nallow if {\n\tinput.ingested.data == \"bar\"\n}",
				},
			},
			out: &interfaces.EvaluationResult{
				Output: "denied",
			},
		},
		{
			name: "Rego deny with extra output",
			eval: &pb.RuleType_Definition_Eval{
				Type: "rego",
				Rego: &pb.RuleType_Definition_Eval_Rego{
					Type: rego.DenyByDefaultEvaluationType.String(),
					Def:  "package minder\n\nimport rego.v1\n\nallow := false\noutput := [\"always fail\", \"never pass\"]",
				},
			},
			out: &interfaces.EvaluationResult{
				Output: []any{"always fail", "never pass"},
			},
		},
		{
			name: "Rego constraints",
			eval: &pb.RuleType_Definition_Eval{
				Type: "rego",
				Rego: &pb.RuleType_Definition_Eval_Rego{
					Type: rego.ConstraintsEvaluationType.String(),
					Def:  "package minder\n\nimport rego.v1\n\nviolations contains results if {\n\tinput.ingested.data == \"foo\"\n\tresults := {\"status\": \"denied\", \"msg\": \"foo is not allowed\"}\n}",
				},
			},
			out: &interfaces.EvaluationResult{
				Output: []any{"foo is not allowed"},
			},
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := &pb.RuleType{
				Def: &pb.RuleType_Definition{
					Eval: tt.eval,
				},
			}

			got, err := eval.NewRuleEvaluator(context.Background(), rt, nil)
			assert.NoError(t, err, "unexpected error")
			assert.NotNil(t, got, "unexpected nil")

			profileData := map[string]any{
				"data": "nothing",
			}
			data := &interfaces.Ingested{
				Object: map[string]any{
					"data": "foo",
				},
			}
			result, err := got.Eval(context.Background(), profileData, nil, data)
			assert.Error(t, err, "expected failure during evaluation")
			assert.Equal(t, tt.out, result)
		})
	}
}

func TestNewRuleEvaluatorWithRegoVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		evalType    string
		def         string
		version     ast.RegoVersion
		wantOutput  any
		wantEvalErr bool
	}{
		{
			name:     "V1 deny-by-default",
			evalType: rego.DenyByDefaultEvaluationType.String(),
			def: `package minder

default allow := false

allow if {
	input.ingested.data == "bar"
}`,
			version:     ast.RegoV1,
			wantOutput:  "denied",
			wantEvalErr: true,
		},
		{
			name:     "V1 constraints",
			evalType: rego.ConstraintsEvaluationType.String(),
			def: `package minder

violations contains result if {
	input.ingested.data == "foo"
	result := {"msg": "foo is not allowed"}
}`,
			version:     ast.RegoV1,
			wantOutput:  []any{"foo is not allowed"},
			wantEvalErr: true,
		},
		{
			name:     "V0 regression",
			evalType: rego.DenyByDefaultEvaluationType.String(),
			def: `package minder

default allow = false

allow {
	input.ingested.data == "bar"
}`,
			version:     ast.RegoV0,
			wantOutput:  "denied",
			wantEvalErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := &pb.RuleType{
				Def: &pb.RuleType_Definition{
					Eval: &pb.RuleType_Definition_Eval{
						Type: "rego",
						Rego: &pb.RuleType_Definition_Eval_Rego{
							Type: tt.evalType,
							Def:  tt.def,
						},
					},
				},
			}

			evaluator, err := eval.NewRuleEvaluator(
				context.Background(),
				rt,
				nil,
				rego.WithRegoVersion(tt.version),
			)
			require.NoError(t, err)
			require.NotNil(t, evaluator)

			result, err := evaluator.Eval(
				context.Background(),
				nil,
				nil,
				&interfaces.Ingested{Object: map[string]any{"data": "foo"}},
			)
			if tt.wantEvalErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.wantOutput, result.Output)
		})
	}
}

func TestV1RuleEvaluatorAllowsMatchingInput(t *testing.T) {
	t.Parallel()

	rt := &pb.RuleType{
		Def: &pb.RuleType_Definition{
			Eval: &pb.RuleType_Definition_Eval{
				Type: "rego",
				Rego: &pb.RuleType_Definition_Eval_Rego{
					Type: rego.DenyByDefaultEvaluationType.String(),
					Def: `package minder

default allow := false

allow if {
	input.ingested.data == "bar"
}`,
				},
			},
		},
	}

	evaluator, err := eval.NewRuleEvaluator(
		context.Background(),
		rt,
		nil,
		rego.WithRegoVersion(ast.RegoV1),
	)
	require.NoError(t, err)
	require.NotNil(t, evaluator)

	result, err := evaluator.Eval(
		context.Background(),
		nil,
		nil,
		&interfaces.Ingested{Object: map[string]any{"data": "bar"}},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Nil(t, result.Output)
}

func TestNewRuleEvaluatorFails(t *testing.T) {
	t.Parallel()

	type args struct {
		rt *pb.RuleType
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "missing eval",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{},
				},
			},
		},
		{
			name: "unexpected engine",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Eval: &pb.RuleType_Definition_Eval{
							Type: "unexpected",
						},
					},
				},
			},
		},
		{
			name: "missing jq",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Eval: &pb.RuleType_Definition_Eval{
							Type: "jq",
						},
					},
				},
			},
		},
		{
			name: "missing jq profile accessor",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Eval: &pb.RuleType_Definition_Eval{
							Type: "jq",
							Jq: []*pb.RuleType_Definition_Eval_JQComparison{
								{
									Ingested: &pb.RuleType_Definition_Eval_JQComparison_Operator{
										Def: ".",
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

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := eval.NewRuleEvaluator(context.Background(), tt.args.rt, nil)
			assert.Error(t, err, "should have errored")
			assert.Nil(t, got, "should be nil")
		})
	}
}
