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

package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/engine"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	defaultProvider = "github"
)

func compareProfiles(t *testing.T, a *minderv1.Profile, b *minderv1.Profile) {
	t.Helper()

	require.Equal(t, a.Name, b.Name, "profile names should match")
	require.Equal(t, a.Context, b.Context, "profile contexts should match")
	compareEntityRules(t, a.Repository, b.Repository)
	compareEntityRules(t, a.BuildEnvironment, b.BuildEnvironment)
	compareEntityRules(t, a.Artifact, b.Artifact)
}

func compareEntityRules(t *testing.T, a []*minderv1.Profile_Rule, b []*minderv1.Profile_Rule) {
	t.Helper()

	require.Equal(t, len(a), len(b), "rule sets should have the same length")

	for i := range a {
		compareRule(t, a[i], b[i])
	}
}

func compareRule(t *testing.T, a *minderv1.Profile_Rule, b *minderv1.Profile_Rule) {
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

	tests := []struct {
		name    string
		profile string
		want    *minderv1.Profile
		wantErr bool
		errIs   error
	}{
		{
			name: "valid",
			profile: `
---
version: v1
type: profile
name: acme-github-profile
context:
  organization: ACME
  provider: github
repository:
  - type: secret_scanning
    def:
      enabled: true
build_environment:
  - type: no_org_wide_github_action_permissions
    def:
      enabled: true
artifact:
  - type: ctlog_entry
    params:
      rekor: 'https://rekor.acme.dev/'
      fulcio: 'https://fulcio.acme.dev/'
      tuf: 'https://tuf.acme.dev/'
    def:
      state: exists
`,
			want: &minderv1.Profile{
				Name: "acme-github-profile",
				Context: &minderv1.Context{
					Provider: defaultProvider,
				},
				Repository: []*minderv1.Profile_Rule{
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
				BuildEnvironment: []*minderv1.Profile_Rule{
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
				Artifact: []*minderv1.Profile_Rule{
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
		{
			name: "valid with only repository",
			profile: `
---
version: v1
type: profile
name: acme-github-profile
context:
  organization: ACME
  provider: github
repository:
  - type: secret_scanning
    def:
      enabled: true
`,
			want: &minderv1.Profile{
				Name: "acme-github-profile",
				Context: &minderv1.Context{
					Provider: defaultProvider,
				},
				Repository: []*minderv1.Profile_Rule{
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
		{
			name: "invalid because of bad YAML",
			profile: `
---
version: v1
type: profile
name: acme-github-profile
context:
  organization: ACME
  provider: github
repository:
  - type: secret_scanning
    def: foobar:
`,
			wantErr: true,
		},
		{
			name: "invalid with no definition",
			profile: `
---
version: v1
type: profile
name: acme-github-profile
context:
  organization: ACME
  provider: github
repository:
  - type: secret_scanning
`,
			wantErr: true,
			errIs:   minderv1.ErrValidationFailed,
		},
		{
			name: "invalid with nil rule",
			profile: `
---
version: v1
type: profile
name: acme-github-profile
context:
  organization: ACME
  provider: github
repository:
  - null
  - type: secret_scanning
    def:
      enabled: true
`,
			wantErr: true,
			errIs:   minderv1.ErrValidationFailed,
		},
		{
			name: "invalid with no name",
			profile: `
---
version: v1
type: profile
repository:
  - type: secret_scanning
    def:
      enabled: true
`,
			wantErr: true,
			errIs:   minderv1.ErrValidationFailed,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := strings.NewReader(tt.profile)

			got, err := engine.ParseYAML(r)
			if tt.wantErr {
				require.Error(t, err, "ParseYAML should have errored")
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs, "ParseYAML should have errored with the expected error")
				}
				return
			}
			require.NoError(t, err, "ParseYAML should not have errored")

			compareProfiles(t, tt.want, got)
		})
	}
}

func TestGetRulesForEntity(t *testing.T) {
	t.Parallel()

	pol := &minderv1.Profile{
		Name: "acme-github-profile",
		Context: &minderv1.Context{
			Provider: defaultProvider,
		},
		Repository: []*minderv1.Profile_Rule{
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
		BuildEnvironment: []*minderv1.Profile_Rule{
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
		Artifact: []*minderv1.Profile_Rule{
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
	}

	type args struct {
		p      *minderv1.Profile
		entity minderv1.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    []*minderv1.Profile_Rule
		wantErr bool
	}{
		{
			name: "valid rules for repository",
			args: args{
				p:      pol,
				entity: minderv1.Entity_ENTITY_REPOSITORIES,
			},
			want: []*minderv1.Profile_Rule{
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
		{
			name: "valid rules for build environment",
			args: args{
				p:      pol,
				entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS,
			},
			want: []*minderv1.Profile_Rule{
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
		{
			name: "valid rules for artifacts",
			args: args{
				p:      pol,
				entity: minderv1.Entity_ENTITY_ARTIFACTS,
			},
			want: []*minderv1.Profile_Rule{
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
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := engine.GetRulesForEntity(tt.args.p, tt.args.entity)
			if tt.wantErr {
				require.Error(t, err, "should have gotten error")
				return
			}

			compareEntityRules(t, tt.want, got)
		})
	}
}

func TestFilterRulesForType(t *testing.T) {
	t.Parallel()

	crs := []*minderv1.Profile_Rule{
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
	}

	type args struct {
		cr []*minderv1.Profile_Rule
		rt *minderv1.RuleType
	}
	tests := []struct {
		name    string
		args    args
		wantLen int
		wantErr bool
	}{
		{
			name: "valid filter for secret scanning",
			args: args{
				cr: crs,
				rt: &minderv1.RuleType{
					Name: "secret_scanning",
				},
			},
			wantLen: 1,
		},
		{
			name: "valid filter for no_org_wide_github_action_permissions",
			args: args{
				cr: crs,
				rt: &minderv1.RuleType{
					Name: "no_org_wide_github_action_permissions",
				},
			},
			wantLen: 1,
		},
		{
			name: "valid filter for ctlog_entry",
			args: args{
				cr: crs,
				rt: &minderv1.RuleType{
					Name: "ctlog_entry",
				},
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := []*minderv1.Profile_Rule{}
			err := engine.TraverseRules(tt.args.cr, func(pp *minderv1.Profile_Rule) error {
				if pp.Type == tt.args.rt.Name {
					got = append(got, pp)
				}
				return nil
			})
			if tt.wantErr {
				require.Error(t, err, "should have gotten error")
				return
			}

			require.Equal(t, tt.wantLen, len(got), "should have gotten the expected number of rules")
		})
	}
}
