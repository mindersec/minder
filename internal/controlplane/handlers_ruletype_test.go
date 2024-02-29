// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/db/embedded"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestCreateRuleType(t *testing.T) {
	t.Parallel()

	// We can't use mockdb.NewMockStore because BeginTransaction returns a *sql.Tx,
	// which is not an interface and can't be mocked.
	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	if err != nil {
		t.Fatalf("Error creating fake store: %v", err)
	}

	// Common database setup
	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating project: %v", err)
	}
	provider, err := dbStore.CreateProvider(ctx, db.CreateProviderParams{
		Name:       "github",
		ProjectID:  dbproj.ID,
		Implements: []db.ProviderType{db.ProviderTypeGithub},
		Definition: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating provider: %v", err)
	}

	tests := []struct {
		name     string
		ruletype *minderv1.CreateRuleTypeRequest
		wantErr  string
		result   *minderv1.CreateRuleTypeResponse
	}{{
		name: "Create ruletype with empty name",
		ruletype: &minderv1.CreateRuleTypeRequest{
			RuleType: &minderv1.RuleType{
				Name: "",
			},
		},
		wantErr: `Couldn't create rule: invalid rule type: rule type name is empty`,
	}, {
		name: "Create ruletype with invalid name",
		ruletype: &minderv1.CreateRuleTypeRequest{
			RuleType: &minderv1.RuleType{
				Name: "colon:invalid",
			},
		},
		wantErr: `Couldn't create rule: invalid rule type: rule type name may only contain letters, numbers, hyphens and underscores`,
	}, {
		name: "Create ruletype without definition",
		ruletype: &minderv1.CreateRuleTypeRequest{
			RuleType: &minderv1.RuleType{
				Name: "empty",
			},
		},
		wantErr: `Couldn't create rule: invalid rule type: rule type definition is nil`,
	}, {
		name: "Create ruletype with valid name and content",
		ruletype: &minderv1.CreateRuleTypeRequest{
			RuleType: &minderv1.RuleType{
				Name: "rule_type_1",
				Def: &minderv1.RuleType_Definition{
					InEntity:   string(minderv1.RepositoryEntity),
					RuleSchema: &structpb.Struct{},
					Ingest:     &minderv1.RuleType_Definition_Ingest{},
					Eval:       &minderv1.RuleType_Definition_Eval{},
				},
			},
		},
		result: &minderv1.CreateRuleTypeResponse{
			RuleType: &minderv1.RuleType{
				Name: "rule_type_1",
				Def: &minderv1.RuleType_Definition{
					InEntity:   string(minderv1.RepositoryEntity),
					RuleSchema: &structpb.Struct{},
					Ingest:     &minderv1.RuleType_Definition_Ingest{},
					Eval:       &minderv1.RuleType_Definition_Eval{},
				},
				Severity: &minderv1.Severity{
					Value: minderv1.Severity_VALUE_UNKNOWN,
				},
			},
		},
	}}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.ruletype.GetContext() == nil {
				tc.ruletype.RuleType.Context = &minderv1.Context{
					Project:  proto.String(dbproj.ID.String()),
					Provider: proto.String("github"),
				}
				if tc.result != nil {
					tc.result.GetRuleType().Context = tc.ruletype.GetContext()
				}
			}

			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Project:  engine.Project{ID: dbproj.ID},
				Provider: engine.Provider{Name: "github"},
			})
			s := &Server{
				store:            dbStore,
				profileValidator: profiles.NewValidator(dbStore),
				evt:              &StubEventer{},
			}

			res, err := s.CreateRuleType(ctx, tc.ruletype)
			if tc.wantErr != "" {
				niceErr, ok := err.(*util.NiceStatus)
				if !ok {
					t.Fatalf("Unexpected error type from CreateProfile: %v", err)
				}
				if niceErr.Details != tc.wantErr {
					t.Errorf("CreateProfile() error = %q, wantErr %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error from CreateProfile: %v", err)
			}

			ruletype, err := dbStore.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
				Provider:  provider.Name,
				Name:      tc.ruletype.GetRuleType().GetName(),
				ProjectID: dbproj.ID,
			})
			if err != nil {
				t.Fatalf("Error getting profile: %v", err)
			}

			if tc.result.GetRuleType().GetId() == "" {
				tc.result.GetRuleType().Id = proto.String(ruletype.ID.String())
			}
			// For some reason, comparing these protos directly doesn't seem to work...
			if !proto.Equal(res, tc.result) {
				t.Errorf("CreateRuleType() got = %v, want %v", res, tc.result)
			}
		})
	}
}
