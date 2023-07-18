// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/mediator/internal/engine"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func comparePolicies(t *testing.T, a *pb.PipelinePolicy, b *pb.PipelinePolicy) {
	t.Helper()

	require.Equal(t, a.Name, b.Name, "policy names should match")
	require.Equal(t, a.Context, b.Context, "policy contexts should match")
	compareEntityRules(t, a.Repository, b.Repository)
	compareEntityRules(t, a.BuildEnvironment, b.BuildEnvironment)
	compareEntityRules(t, a.Artifact, b.Artifact)
}

func compareEntityRules(t *testing.T, a []*pb.PipelinePolicy_ContextualRuleSet, b []*pb.PipelinePolicy_ContextualRuleSet) {
	t.Helper()

	require.Equal(t, len(a), len(b), "rule sets should have the same length")

	for i := range a {
		compareRules(t, a[i].Rules, b[i].Rules)
	}
}

func compareRules(t *testing.T, a []*pb.PipelinePolicy_Rule, b []*pb.PipelinePolicy_Rule) {
	t.Helper()

	require.Equal(t, len(a), len(b), "rules should have the same length")

	for i := range a {
		compareRule(t, a[i], b[i])
	}
}

func compareRule(t *testing.T, a *pb.PipelinePolicy_Rule, b *pb.PipelinePolicy_Rule) {
	t.Helper()

	require.Equal(t, a.Type, b.Type, "rule types should match")

	if a.Params != nil {
		require.NotNil(t, b.Params, "rule params should not be nil")
		require.Equal(t, len(a.Params.Fields), len(b.Params.Fields), "rule params should have the same length")
		for k := range a.Params.Fields {
			compareValues(t, a.Params.Fields[k], b.Params.Fields[k])
		}
	} else {
		require.Nil(t, b.Params, "rule params should be nil")
	}

	if a.Def != nil {
		require.NotNil(t, b.Def, "rule defs should not be nil")
		require.Equal(t, len(a.Def.Fields), len(b.Def.Fields), "rule defs should have the same length")

		for k := range a.Def.Fields {
			compareValues(t, a.Def.Fields[k], b.Def.Fields[k])
		}
	} else {
		require.Nil(t, b.Def, "rule defs should be nil")
	}
}

func compareValues(t *testing.T, a *structpb.Value, b *structpb.Value) {
	t.Helper()

	require.Equal(t, a.Kind, b.Kind, "value kinds should match")

	switch a.Kind.(type) {
	case *structpb.Value_StringValue:
		require.Equal(t, a.GetStringValue(), b.GetStringValue(), "string values should match")
	case *structpb.Value_BoolValue:
		require.Equal(t, a.GetBoolValue(), b.GetBoolValue(), "bool values should match")
	case *structpb.Value_NumberValue:
		require.Equal(t, a.GetNumberValue(), b.GetNumberValue(), "number values should match")
	case *structpb.Value_StructValue:
		compareStructs(t, a.GetStructValue(), b.GetStructValue())
	}
}

func compareStructs(t *testing.T, a *structpb.Struct, b *structpb.Struct) {
	t.Helper()

	require.Equal(t, len(a.Fields), len(b.Fields), "struct fields should have the same length")

	for k := range a.Fields {
		compareValues(t, a.Fields[k], b.Fields[k])
	}
}

func TestParseYAML(t *testing.T) {
	t.Parallel()

	defaultOrg := "ACME"
	defaultProvider := "github"
	quayProvider := "quay"

	tests := []struct {
		name    string
		policy  string
		want    *pb.PipelinePolicy
		wantErr bool
		errIs   error
	}{
		{
			name: "valid",
			policy: `
---
version: v1
type: pipeline-policy
name: acme-github-policy
context:
  organization: ACME
  provider: github
repository:
  - context: github
    rules:
      - type: secret_scanning
        def:
          enabled: true
build_environment:
  - rules:  # Not specifying a context takes the default context
      - type: no_org_wide_github_action_permissions
        def:
          enabled: true
artifact:
  - context: quay
    rules:
      - type: ctlog_entry
        params:
          rekor: 'https://rekor.acme.dev/'
          fulcio: 'https://fulcio.acme.dev/'
          tuf: 'https://tuf.acme.dev/'
        def:
          state: exists
`,
			want: &pb.PipelinePolicy{
				Name: "acme-github-policy",
				Context: &pb.Context{
					Organization: &defaultOrg,
					Provider:     "github",
				},
				Repository: []*pb.PipelinePolicy_ContextualRuleSet{
					{
						Context: &defaultProvider,
						Rules: []*pb.PipelinePolicy_Rule{
							{
								Type: "secret_scanning",
								Def: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"enabled": {
											Kind: &structpb.Value_BoolValue{
												BoolValue: true,
											},
										},
									},
								},
							},
						},
					},
				},
				BuildEnvironment: []*pb.PipelinePolicy_ContextualRuleSet{
					{
						Rules: []*pb.PipelinePolicy_Rule{
							{
								Type: "no_org_wide_github_action_permissions",
								Def: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"enabled": {
											Kind: &structpb.Value_BoolValue{
												BoolValue: true,
											},
										},
									},
								},
							},
						},
					},
				},
				Artifact: []*pb.PipelinePolicy_ContextualRuleSet{
					{
						Context: &quayProvider,
						Rules: []*pb.PipelinePolicy_Rule{
							{
								Type: "ctlog_entry",
								Params: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"rekor": {
											Kind: &structpb.Value_StringValue{
												StringValue: "https://rekor.acme.dev/",
											},
										},
										"fulcio": {
											Kind: &structpb.Value_StringValue{
												StringValue: "https://fulcio.acme.dev/",
											},
										},
										"tuf": {
											Kind: &structpb.Value_StringValue{
												StringValue: "https://tuf.acme.dev/",
											},
										},
									},
								},
								Def: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"state": {
											Kind: &structpb.Value_StringValue{
												StringValue: "exists",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "valid with only repository",
			policy: `
---
version: v1
type: pipeline-policy
name: acme-github-policy
context:
  organization: ACME
  provider: github
repository:
  - context: github
    rules:
      - type: secret_scanning
        def:
          enabled: true
`,
			want: &pb.PipelinePolicy{
				Name: "acme-github-policy",
				Context: &pb.Context{
					Organization: &defaultOrg,
					Provider:     "github",
				},
				Repository: []*pb.PipelinePolicy_ContextualRuleSet{
					{
						Context: &defaultProvider,
						Rules: []*pb.PipelinePolicy_Rule{
							{
								Type: "secret_scanning",
								Def: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"enabled": {
											Kind: &structpb.Value_BoolValue{
												BoolValue: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "invalid because of bad YAML",
			policy: `
---
version: v1
type: pipeline-policy
name: acme-github-policy
context:
  organization: ACME
  provider: github
repository:
  - context: github
    rules:
      - type: secret_scanning
        def: foobar:
`,
			wantErr: true,
		},
		{
			name: "invalid with no definition",
			policy: `
---
version: v1
type: pipeline-policy
name: acme-github-policy
context:
  organization: ACME
  provider: github
repository:
  - context: github
    rules:
      - type: secret_scanning
`,
			wantErr: true,
			errIs:   engine.ErrValidationFailed,
		},
		{
			name: "invalid with nil rule",
			policy: `
---
version: v1
type: pipeline-policy
name: acme-github-policy
context:
  organization: ACME
  provider: github
repository:
  - null
  - context: github
    rules:
      - type: secret_scanning
        def:
          enabled: true
`,
			wantErr: true,
			errIs:   engine.ErrValidationFailed,
		},
		{
			name: "invalid with no context",
			policy: `
---
version: v1
type: pipeline-policy
name: acme-github-policy
repository:
  - context: github
    rules:
      - type: secret_scanning
        def:
          enabled: true
`,
			wantErr: true,
			errIs:   engine.ErrValidationFailed,
		},
		{
			name: "invalid with no name",
			policy: `
---
version: v1
type: pipeline-policy
repository:
  - context: github
    rules:
      - type: secret_scanning
        def:
          enabled: true
`,
			wantErr: true,
			errIs:   engine.ErrValidationFailed,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := strings.NewReader(tt.policy)

			got, err := engine.ParseYAML(r)
			if tt.wantErr {
				require.Error(t, err, "ParseYAML should have errored")
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs, "ParseYAML should have errored with the expected error")
				}
				return
			}
			require.NoError(t, err, "ParseYAML should not have errored")

			comparePolicies(t, tt.want, got)
		})
	}
}
