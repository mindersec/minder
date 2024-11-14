// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestRuleType_Definition_Eval_JQComparison_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jq      *RuleType_Definition_Eval_JQComparison
		wantErr bool
	}{
		{
			name: "valid JQComparison",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
				Profile: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
			},
			wantErr: false,
		},
		{
			name:    "nil JQComparison",
			jq:      nil,
			wantErr: true,
		},
		{
			name: "empty ingested definition",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: "",
				},
			},
			wantErr: true,
		},
		{
			name: "profile and constant accessors both present",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
				Profile: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
				Constant: structpb.NewStringValue("constant definition"),
			},
			wantErr: true,
		},
		{
			name: "missing profile or constant accessor",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
			},
			wantErr: true,
		},
		{
			name: "empty profile accessor definition",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
				Profile: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: "",
				},
			},
			wantErr: true,
		},
		{
			name: "unparsable ingested definition",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".foo[",
				},
				Profile: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ingested definition",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: "invalid",
				},
				Profile: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
			},
			wantErr: true,
		},
		{
			name: "unparsable profile accessor definition",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
				Profile: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".foo[",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid profile accessor definition",
			jq: &RuleType_Definition_Eval_JQComparison{
				Ingested: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: ".",
				},
				Profile: &RuleType_Definition_Eval_JQComparison_Operator{
					Def: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.jq.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleType_Definition_Eval_Rego_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rego    *RuleType_Definition_Eval_Rego
		wantErr bool
	}{
		{
			name: "valid rego definition",
			rego: &RuleType_Definition_Eval_Rego{
				Def: "package example.policy\n\nallow { true }",
			},
			wantErr: false,
		},
		{
			name:    "nil rego",
			rego:    nil,
			wantErr: true,
		},
		{
			name: "empty rego definition",
			rego: &RuleType_Definition_Eval_Rego{
				Def: "",
			},
			wantErr: true,
		},
		{
			name: "invalid syntax rego definition",
			rego: &RuleType_Definition_Eval_Rego{
				Def: "package example.policy\n\nallow {",
			},
			wantErr: true,
		},
		{
			name: "missing import rego definition",
			rego: &RuleType_Definition_Eval_Rego{
				Def: "package example.policy\n\nallow if { input.ingested.url != \"\" }",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.rego.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
