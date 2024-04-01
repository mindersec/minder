// Copyright 2024 Stacklok, Inc
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
	"reflect"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/db/embedded"
	"github.com/stacklok/minder/internal/engine"
	stubeventer "github.com/stacklok/minder/internal/events/stubs"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestGetUnusedOldRuleTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		newRules   profiles.RuleMapping
		oldRules   profiles.RuleMapping
		wantUnused []profiles.EntityAndRuleTuple
	}{
		{
			name: "Unused rule in oldRules",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
			wantUnused: []profiles.EntityAndRuleTuple{
				{Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
		},
		{
			name: "Multiple unused rules in oldRules",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
				{RuleType: "Type4", RuleName: "Name4"}: {Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type4", "Name4")},
			},
			wantUnused: []profiles.EntityAndRuleTuple{
				{Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
				{Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type4", "Name4")},
			},
		},
		{
			name: "No unused rules in oldRules",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			wantUnused: nil,
		},
		{
			name: "Unused rules with same rule type",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "Name3")},
			},
			// All rule types are used
			wantUnused: nil,
		},
		{
			name: "Unused rules with same rule type but different entity types",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "Name3")},
			},
			// All rule types are used
			wantUnused: nil,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			unusedRuleTypes := getUnusedOldRuleTypes(test.newRules, test.oldRules)
			require.ElementsMatch(t, test.wantUnused, unusedRuleTypes)
		})
	}
}

func TestGetUnusedOldRuleStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		newRules        profiles.RuleMapping
		oldRules        profiles.RuleMapping
		wantUnusedRules profiles.RuleMapping
	}{
		{
			name: "Unused rule in oldRules",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
			wantUnusedRules: profiles.RuleMapping{
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
		},
		{
			name: "No unused rules in oldRules",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			wantUnusedRules: profiles.RuleMapping{},
		},
		{
			name: "Unused rules with same rule type",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "Name3")},
			},
			wantUnusedRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "Name3")},
			},
		},
		{
			name: "Unused old rules statuses with empty name",
			newRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: ""}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "")},
				{RuleType: "Type2", RuleName: ""}: {Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type2", "")},
			},
			wantUnusedRules: profiles.RuleMapping{
				{RuleType: "Type1", RuleName: ""}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "")},
				{RuleType: "Type2", RuleName: ""}: {Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type2", "")},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotUnusedRules := getUnusedOldRuleStatuses(test.newRules, test.oldRules)
			require.True(t, reflect.DeepEqual(test.wantUnusedRules, gotUnusedRules))
		})
	}
}

//nolint:gocyclo
func TestCreateProfile(t *testing.T) {
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

	// The provider is used in the profile definition
	_, err = dbStore.CreateProvider(ctx, db.CreateProviderParams{
		Name:       "github",
		ProjectID:  dbproj.ID,
		Class:      db.NullProviderClass{ProviderClass: db.ProviderClassGithub, Valid: true},
		Implements: []db.ProviderType{db.ProviderTypeGithub},
		AuthFlows:  []db.AuthorizationFlow{db.AuthorizationFlowUserInput},
		Definition: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating provider: %v", err)
	}
	_, err = dbStore.CreateRuleType(ctx, db.CreateRuleTypeParams{
		Name:          "rule_type_1",
		ProjectID:     dbproj.ID,
		Definition:    []byte(`{"in_entity": "repository","ruleSchema":{}}`),
		SeverityValue: db.SeverityLow,
	})
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	tests := []struct {
		name    string
		profile *minderv1.CreateProfileRequest
		wantErr string
		result  *minderv1.CreateProfileResponse
	}{{
		name: "Create profile with empty name",
		profile: &minderv1.CreateProfileRequest{
			Profile: &minderv1.Profile{
				Name: "",
			},
		},
		wantErr: `Couldn't create profile: validation failed: profile name cannot be empty`,
	}, {
		name: "Create profile with invalid name",
		profile: &minderv1.CreateProfileRequest{
			Profile: &minderv1.Profile{
				Name: "colon:invalid",
			},
		},
		wantErr: `Couldn't create profile: validation failed: name may only contain letters, numbers, hyphens and underscores`,
	}, {
		name: "Create profile with no rules",
		profile: &minderv1.CreateProfileRequest{
			Profile: &minderv1.Profile{
				Name: "test",
			},
		},
		wantErr: `Couldn't create profile: validation failed: profile must have at least one rule`,
	},
		{
			name: "Create profile with valid name and rules",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test",
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
			result: &minderv1.CreateProfileResponse{
				Profile: &minderv1.Profile{
					Name:      "test",
					Alert:     proto.String("on"),
					Remediate: proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Name: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
		},
		{
			name: "Create profile with explicit alert and remediate",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_explicit",
					Alert:     proto.String("off"),
					Remediate: proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
			result: &minderv1.CreateProfileResponse{
				Profile: &minderv1.Profile{
					Name:      "test_explicit",
					Alert:     proto.String("off"),
					Remediate: proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Name: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.profile.GetContext() == nil {
				tc.profile.Profile.Context = &minderv1.Context{
					Project:  proto.String(dbproj.ID.String()),
					Provider: proto.String("github"),
				}
				if tc.result != nil {
					tc.result.GetProfile().Context = tc.profile.GetContext()
				}
			}

			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Project:  engine.Project{ID: dbproj.ID},
				Provider: engine.Provider{Name: "github"},
			})
			evts := &stubeventer.StubEventer{}
			s := &Server{
				store: dbStore,
				// Do not replace this with a mock - these tests are used to test ProfileService as well
				profiles: profiles.NewProfileService(evts),
				evt:      evts,
			}

			res, err := s.CreateProfile(ctx, tc.profile)
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

			profile, err := dbStore.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
				Name:      tc.profile.GetProfile().GetName(),
				ProjectID: dbproj.ID,
			})
			if err != nil {
				t.Fatalf("Error getting profile: %v", err)
			}

			if tc.result.GetProfile().GetId() == "" {
				tc.result.GetProfile().Id = proto.String(profile.ID.String())
			}
			// For some reason, comparing these protos directly doesn't seem to work...
			if !proto.Equal(res, tc.result) {
				t.Errorf("CreateProfile() got = %v, want %v", res, tc.result)
			}
		})
	}
}

func generateConsistentUUID(t *testing.T, ruleType, ruleName string) uuid.UUID {
	t.Helper()
	return uuid.NewSHA1(uuid.Nil, []byte(ruleType+ruleName))
}
