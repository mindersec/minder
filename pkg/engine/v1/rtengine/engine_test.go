// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rtengine

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/util/ptr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

func TestGitProvider(t *testing.T) {
	t.Parallel()

	type ruleInstance struct {
		Def    map[string]any
		Params map[string]any
	}
	tests := []struct {
		name     string
		ent      protoreflect.ProtoMessage
		ruleType *minderv1.RuleType
		ri       ruleInstance
		wantErr  bool
		dirSetup func(t *testing.T, tdir string)
	}{
		{
			name: "simple",
			ent: &minderv1.Repository{
				CloneUrl: "foo",
			},
			ruleType: &minderv1.RuleType{
				Context: &minderv1.Context{
					Project: ptr.Ptr("test"),
				},
				Def: &minderv1.RuleType_Definition{
					InEntity:   minderv1.RepositoryEntity.String(),
					RuleSchema: &structpb.Struct{},
					Ingest: &minderv1.RuleType_Definition_Ingest{
						Type: "git",
					},
					Eval: &minderv1.RuleType_Definition_Eval{
						Type: "rego",
						Rego: &minderv1.RuleType_Definition_Eval_Rego{
							Type: "deny-by-default",
							Def: `package minder

import rego.v1

default allow := false

allow if {
	file.exists("README.md")
}
`,
						},
					},
				},
			},
			ri: ruleInstance{
				Def:    map[string]any{},
				Params: nil,
			},
			wantErr: false,
			dirSetup: func(t *testing.T, tdir string) {
				t.Helper()

				err := os.WriteFile(tdir+"/README.md", []byte("hello"), 0600)
				require.NoError(t, err, "os.WriteFile() failed")
			},
		},
		{
			name: "nil ruleDef treated as empty map",
			ent: &minderv1.Repository{
				CloneUrl: "foo",
			},
			ruleType: &minderv1.RuleType{
				Context: &minderv1.Context{
					Project: ptr.Ptr("test"),
				},
				Def: &minderv1.RuleType_Definition{
					InEntity:   minderv1.RepositoryEntity.String(),
					RuleSchema: &structpb.Struct{},
					Ingest: &minderv1.RuleType_Definition_Ingest{
						Type: "git",
					},
					Eval: &minderv1.RuleType_Definition_Eval{
						Type: "rego",
						Rego: &minderv1.RuleType_Definition_Eval_Rego{
							Type: "deny-by-default",
							Def: `package minder

import rego.v1

default allow := false

allow if {
	file.exists("README.md")
}
`,
						},
					},
				},
			},
			ri: ruleInstance{
				Def:    nil,
				Params: nil,
			},
			wantErr: false,
			dirSetup: func(t *testing.T, tdir string) {
				t.Helper()

				err := os.WriteFile(tdir+"/README.md", []byte("hello"), 0600)
				require.NoError(t, err, "os.WriteFile() failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// set up zerolog's test logger
			tlw := zerolog.NewTestWriter(t)
			ctx := zerolog.New(tlw).With().Logger().WithContext(context.Background())

			tdir := t.TempDir()

			if tt.dirSetup != nil {
				tt.dirSetup(t, tdir)
			}

			tk := tkv1.NewTestKit(tkv1.WithGitDir(tdir))
			rte, err := NewRuleTypeEngine(ctx, tt.ruleType, tk)
			require.NoError(t, err, "NewRuleTypeEngine() failed")

			// Override ingester. This is needed for the test.
			rte.WithCustomIngester(tk)

			_, err = rte.Eval(ctx, tt.ent, tt.ri.Def, tt.ri.Params, tkv1.NewVoidResultSink())
			if tt.wantErr {
				assert.Error(t, err, "Eval() should have failed")
			} else {
				assert.NoError(t, err, "Eval() failed")
			}
		})
	}
}

func TestNewRuleTypeEngineProviderTraits(t *testing.T) {
	t.Parallel()

	newRuleType := func(traits ...string) *minderv1.RuleType {
		return &minderv1.RuleType{
			Name: "test-rule",
			Context: &minderv1.Context{
				Project: ptr.Ptr("test"),
			},
			Def: &minderv1.RuleType_Definition{
				InEntity:       minderv1.RepositoryEntity.String(),
				RuleSchema:     &structpb.Struct{},
				ProviderTraits: traits,
				Ingest: &minderv1.RuleType_Definition_Ingest{
					Type: "git",
				},
				Eval: &minderv1.RuleType_Definition_Eval{
					Type: "rego",
					Rego: &minderv1.RuleType_Definition_Eval_Rego{
						Type: "deny-by-default",
						Def: `package minder

import rego.v1

default allow := false
`,
					},
				},
			},
		}
	}

	tests := []struct {
		name          string
		ruleType      *minderv1.RuleType
		canImplement  func(trait minderv1.ProviderType) bool
		wantSupported bool
	}{
		{
			name:     "no provider_traits is unaffected",
			ruleType: newRuleType(),
			canImplement: func(minderv1.ProviderType) bool {
				return false
			},
			wantSupported: true,
		},
		{
			name:     "provider satisfies all required traits",
			ruleType: newRuleType("git", "github"),
			canImplement: func(minderv1.ProviderType) bool {
				return true
			},
			wantSupported: true,
		},
		{
			name:     "provider missing a required trait",
			ruleType: newRuleType("git", "github"),
			canImplement: func(trait minderv1.ProviderType) bool {
				return trait != minderv1.ProviderType_PROVIDER_TYPE_GITHUB
			},
			wantSupported: false,
		},
		{
			// Defensive path: rule type creation validates provider_traits
			// (see ruletypes.validateProviderTraits), so this shouldn't
			// happen for rule types created after that validation existed.
			// It's a fallback for rule types stored before then, or any
			// path that bypasses validation — not intended behaviour. The
			// engine also logs a warning here, which this test doesn't
			// assert on directly.
			name:     "unknown trait name is a defensive fallback, treated as unsupported and warned about",
			ruleType: newRuleType("not-a-real-trait"),
			canImplement: func(minderv1.ProviderType) bool {
				return true
			},
			wantSupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tlw := zerolog.NewTestWriter(t)
			ctx := zerolog.New(tlw).With().Logger().WithContext(context.Background())

			tk := tkv1.NewTestKit(tkv1.WithGitDir(t.TempDir()), tkv1.WithCanImplement(tt.canImplement))
			rte, err := NewRuleTypeEngine(ctx, tt.ruleType, tk)
			require.NoError(t, err, "NewRuleTypeEngine() failed")
			assert.Equal(t, tt.wantSupported, rte.SupportedByProvider())
		})
	}
}
