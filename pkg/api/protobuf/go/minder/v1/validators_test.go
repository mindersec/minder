// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestRuleType_Definition_Ingest_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ingest  *RuleType_Definition_Ingest
		wantErr bool
	}{
		{
			name: "valid diff ingest",
			ingest: &RuleType_Definition_Ingest{
				Type: IngestTypeDiff,
				Diff: &DiffType{},
			},
			wantErr: false,
		},
		{
			name: "valid rest ingest",
			ingest: &RuleType_Definition_Ingest{
				Type: "rest",
				Rest: &RestType{
					Endpoint: "https://example.com/api",
				},
			},
			wantErr: false,
		},
		{
			name:    "nil ingest",
			ingest:  nil,
			wantErr: true,
		},
		{
			name: "invalid diff ingest",
			ingest: &RuleType_Definition_Ingest{
				Type: IngestTypeDiff,
				Diff: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid rest ingest",
			ingest: &RuleType_Definition_Ingest{
				Type: "rest",
				Rest: &RestType{
					Endpoint: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ingest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleType_Definition_Eval_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		eval    *RuleType_Definition_Eval
		wantErr bool
	}{
		{
			name: "valid eval definition",
			eval: &RuleType_Definition_Eval{
				Type: "rego",
				Rego: &RuleType_Definition_Eval_Rego{
					Def: "package example.policy\n\nallow { true }",
				},
				DataSources: []*DataSourceReference{
					{
						Name:  "osv",
						Alias: "osv_data",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "eval definition with duplicate alias",
			eval: &RuleType_Definition_Eval{
				Type: "rego",
				Rego: &RuleType_Definition_Eval_Rego{
					Def: "package example.policy\n\nallow { true }",
				},
				DataSources: []*DataSourceReference{
					{
						Name:  "osv1",
						Alias: "osv_data",
					},
					{
						Name:  "osv2",
						Alias: "osv_data",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "eval definition with same name and alias",
			eval: &RuleType_Definition_Eval{
				Rego: &RuleType_Definition_Eval_Rego{
					Def: "package example.policy\n\nallow { true }",
				},
				DataSources: []*DataSourceReference{
					{
						Name: "osv_data",
					},
					{
						Name:  "osv",
						Alias: "osv_data",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.eval.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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

func TestDataSourceReference_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ds      *DataSourceReference
		wantErr bool
	}{
		{
			name: "valid data source reference with alias",
			ds: &DataSourceReference{
				Name:  "namespace/name",
				Alias: "my_data_source",
			},
			wantErr: false,
		},
		{
			name: "valid data source reference without alias",
			ds: &DataSourceReference{
				Name: "osv",
			},
			wantErr: false,
		},
		{
			name:    "no name",
			ds:      &DataSourceReference{},
			wantErr: true,
		},
		{
			name: "no alias and name with invalid characters",
			ds: &DataSourceReference{
				Name: "invalid/name",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ds.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleType_Definition_Alert_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		alert   *RuleType_Definition_Alert
		wantErr bool
	}{
		{
			name: "valid alert definition",
			alert: &RuleType_Definition_Alert{
				Type:             "security_advisory",
				SecurityAdvisory: &RuleType_Definition_Alert_AlertTypeSA{},
			},
			wantErr: false,
		},
		{
			name:    "nil alert is valid",
			alert:   nil,
			wantErr: false,
		},
		{
			name: "empty alert type",
			alert: &RuleType_Definition_Alert{
				Type: "",
			},
			wantErr: true,
		},
		{
			name: "invalid security advisory",
			alert: &RuleType_Definition_Alert{
				Type:             "security_advisory",
				SecurityAdvisory: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.alert.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleType_Definition_Alert_AlertTypePRComment_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		prAlert *RuleType_Definition_Alert_AlertTypePRComment
		wantErr bool
	}{
		{
			name: "valid PR comment alert",
			prAlert: &RuleType_Definition_Alert_AlertTypePRComment{
				ReviewMessage: "This is a PR comment",
			},
			wantErr: false,
		},
		{
			name: "valid PR comment alert template",
			prAlert: &RuleType_Definition_Alert_AlertTypePRComment{
				ReviewMessage: "This is a PR comment with a template {{ .EvalErrorDetails }}",
			},
			wantErr: false,
		},
		{
			name: "unparsable PR comment alert template",
			prAlert: &RuleType_Definition_Alert_AlertTypePRComment{
				ReviewMessage: "{{ ",
			},
			wantErr: true,
		},
		{
			name:    "empty PR comment message is invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.prAlert.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleType_Definition_Remediate_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rem     *RuleType_Definition_Remediate
		wantErr bool
	}{
		{
			name: "valid rest remediation",
			rem: &RuleType_Definition_Remediate{
				Type: "rest",
				Rest: &RestType{
					Endpoint: "https://example.com/api",
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull request remediation",
			rem: &RuleType_Definition_Remediate{
				Type: "pull_request",
				PullRequest: &RuleType_Definition_Remediate_PullRequestRemediation{
					Title: "Fix issue",
					Body:  "This PR fixes the issue.",
				},
			},
			wantErr: false,
		},
		{
			name: "valid GitHub branch protection remediation",
			rem: &RuleType_Definition_Remediate{
				Type: "gh_branch_protection",
				GhBranchProtection: &RuleType_Definition_Remediate_GhBranchProtectionType{
					Patch: "patch content",
				},
			},
			wantErr: false,
		},
		{
			name:    "nil remediation",
			rem:     nil,
			wantErr: false,
		},
		{
			name: "empty remediation type",
			rem: &RuleType_Definition_Remediate{
				Type: "",
			},
			wantErr: true,
		},
		{
			name: "multiple remediation data",
			rem: &RuleType_Definition_Remediate{
				Type: "rest",
				Rest: &RestType{
					Endpoint: "https://example.com/api",
				},
				GhBranchProtection: &RuleType_Definition_Remediate_GhBranchProtectionType{
					Patch: "patch content",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.rem.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRestType_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rest    *RestType
		wantErr bool
	}{
		{
			name: "valid rest remediation",
			rest: &RestType{
				Endpoint: "https://example.com/api",
			},
			wantErr: false,
		},
		{
			name: "empty remediation endpoint",
			rest: &RestType{
				Endpoint: "",
			},
			wantErr: true,
		},
		{
			name: "templated method",
			rest: &RestType{
				Method:   "{{if .CanPut}}PUT{{else}}POST{{end}}",
				Endpoint: "https://example.com/api",
			},
			wantErr: false,
		},
		{
			name: "templated URI",
			rest: &RestType{
				Endpoint: "https://example.com/api/{{if .CanPut}}object{{end}}",
			},
			wantErr: false,
		},
		{
			name: "invalid templated method",
			rest: &RestType{
				Method:   "{{if .CanPut}}PUT",
				Endpoint: "https://example.com/api",
			},
			wantErr: true,
		},
		{
			name: "invalid templated endpoint",
			rest: &RestType{
				Endpoint: "https://example.com/api{{else}}object{{end}}",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.rest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleType_Definition_Remediate_PullRequestRemediation_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		prRem   *RuleType_Definition_Remediate_PullRequestRemediation
		wantErr bool
	}{
		{
			name: "valid pull request remediation",
			prRem: &RuleType_Definition_Remediate_PullRequestRemediation{
				Title: "Fix issue",
				Body:  "This pull request adds a Dependabot configuration to the repository to handle package updates for {{.Profile.package_ecosystem }}.",
			},
			wantErr: false,
		},
		{
			name: "empty pull request title",
			prRem: &RuleType_Definition_Remediate_PullRequestRemediation{
				Title: "",
				Body:  "This pull request adds a Dependabot configuration to the repository to handle package updates for {{.Profile.package_ecosystem }}.",
			},
			wantErr: true,
		},
		{
			name: "empty pull request body",
			prRem: &RuleType_Definition_Remediate_PullRequestRemediation{
				Title: "Fix issue",
				Body:  "",
			},
			wantErr: true,
		},
		{
			name: "malformed pull request body",
			prRem: &RuleType_Definition_Remediate_PullRequestRemediation{
				Title: "Fix issue",
				Body:  "{{ .Name",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.prRem.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleType_Definition_Remediate_GhBranchProtectionType_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ghp     *RuleType_Definition_Remediate_GhBranchProtectionType
		wantErr bool
	}{
		{
			name: "valid GitHub branch protection remediation",
			ghp: &RuleType_Definition_Remediate_GhBranchProtectionType{
				Patch: "patch content",
			},
			wantErr: false,
		},
		{
			name: "empty branch protection patch template",
			ghp: &RuleType_Definition_Remediate_GhBranchProtectionType{
				Patch: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ghp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
