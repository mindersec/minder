// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package eval provides necessary interfaces and implementations for evaluating
// rules.
package eval_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/internal/engine/eval"
	"github.com/mindersec/minder/internal/engine/eval/rego"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewRuleEvaluatorWorks(t *testing.T) {
	t.Parallel()

	type args struct {
		rt *pb.RuleType
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "JQ",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Eval: &pb.RuleType_Definition_Eval{
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
					},
				},
			},
		},
		{
			name: "Rego",
			args: args{
				rt: &pb.RuleType{
					Def: &pb.RuleType_Definition{
						Eval: &pb.RuleType_Definition_Eval{
							Type: "rego",
							Rego: &pb.RuleType_Definition_Eval_Rego{
								Type: rego.DenyByDefaultEvaluationType.String(),
								Def:  "package minder\n\ndefault allow = false\n\nallow {\n\tinput.ingested.data == \"foo\"\n}",
							},
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

			got, err := eval.NewRuleEvaluator(context.Background(), tt.args.rt, nil, nil)
			assert.NoError(t, err, "unexpected error")
			assert.NotNil(t, got, "unexpected nil")
		})
	}
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := eval.NewRuleEvaluator(context.Background(), tt.args.rt, nil, nil)
			assert.Error(t, err, "should have errored")
			assert.Nil(t, got, "should be nil")
		})
	}
}
