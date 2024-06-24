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
	"fmt"
	"reflect"
	"strings"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/db/embedded"
	"github.com/stacklok/minder/internal/engine/engcontext"
	stubeventer "github.com/stacklok/minder/internal/events/stubs"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/providers"
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
		wantErr: `Couldn't create profile: validation failed: name may only contain letters, numbers, hyphens and underscores, and is limited to a maximum of 63 characters`,
	}, {
		name: "Create profile with no rules",
		profile: &minderv1.CreateProfileRequest{
			Profile: &minderv1.Profile{
				Name: "test_norules",
			},
		},
		result: &minderv1.CreateProfileResponse{
			Profile: &minderv1.Profile{
				Name:      "test_norules",
				Alert:     proto.String("on"),
				Remediate: proto.String("off"),
			},
		},
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
		{
			name: "Create profile with explicit display name",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:        "test_explicit_display",
					DisplayName: "This is an explicit display name",
					Alert:       proto.String("off"),
					Remediate:   proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
			result: &minderv1.CreateProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_explicit_display",
					DisplayName: "This is an explicit display name",
					Alert:       proto.String("off"),
					Remediate:   proto.String("on"),
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
					Project: proto.String(dbproj.ID.String()),
				}
				if tc.result != nil {
					tc.result.GetProfile().Context = tc.profile.GetContext()
				}
			}

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})
			evts := &stubeventer.StubEventer{}
			s := &Server{
				store: dbStore,
				// Do not replace this with a mock - these tests are used to test ProfileService as well
				profiles:      profiles.NewProfileService(evts),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
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

func ruleTypeName(prefix string, num int) string {
	return fmt.Sprintf("%s_rule_type_%d", prefix, num)
}

func ruleTypeSequenceCreate(
	ctx context.Context,
	dbStore db.Store,
	entity minderv1.Entity,
	projectID uuid.UUID,
	prefix string,
	count int,
) error {
	defstr := fmt.Sprintf("{\"in_entity\": \"%s\",\"ruleSchema\":{}}", entity.ToString())

	for i := 0; i < count; i++ {
		_, err := dbStore.CreateRuleType(ctx, db.CreateRuleTypeParams{
			Name:          ruleTypeName(prefix, i+1),
			ProjectID:     projectID,
			Definition:    []byte(defstr),
			SeverityValue: db.SeverityLow,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// nolint:gocyclo
func TestPatchProfile(t *testing.T) {
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

	err = ruleTypeSequenceCreate(ctx, dbStore, minderv1.Entity_ENTITY_REPOSITORIES, dbproj.ID, "repo", 4)
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	err = ruleTypeSequenceCreate(ctx, dbStore, minderv1.Entity_ENTITY_PULL_REQUESTS, dbproj.ID, "pull", 4)
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	err = ruleTypeSequenceCreate(ctx, dbStore, minderv1.Entity_ENTITY_ARTIFACTS, dbproj.ID, "artifact", 4)
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	tests := []struct {
		name         string
		baseProfile  *minderv1.CreateProfileRequest
		patchRequest *minderv1.PatchProfileRequest
		wantErr      string
		result       *minderv1.PatchProfileResponse
	}{
		{
			name: "Patch profile with change to remediate",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_rem_patch",
					Remediate: proto.String("off"),
					Alert:     proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"remediate"}},
				Patch: &minderv1.Profile{
					Remediate: proto.String("on"),
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch",
					Remediate:   proto.String("on"),
					Alert:       proto.String("off"),
					DisplayName: "test_rem_patch",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with change to alert",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_rem_patch_alert",
					Remediate: proto.String("on"),
					Alert:     proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"alert"}},
				Patch: &minderv1.Profile{
					Alert: proto.String("off"),
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_alert",
					Remediate:   proto.String("on"),
					Alert:       proto.String("off"),
					DisplayName: "test_rem_patch_alert",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with repo rule addition",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_patch_repo_rule_add",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"repository"}},
				Patch: &minderv1.Profile{
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_add",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_rem_patch_repo_rule_add",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 2),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with repo rule deletion",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_patch_repo_rule_del",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"repository"}},
				Patch: &minderv1.Profile{
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_del",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_rem_patch_repo_rule_del",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with all allowed replacements",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace_all",
					Remediate:   proto.String("off"),
					Alert:       proto.String("off"),
					DisplayName: "display_test_rem_patch_repo_rule_replace_all",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
					Artifact: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("artifact", 3),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("artifact", 4),
							Def:  &structpb.Struct{},
						},
					},
					PullRequest: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("pull", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("pull", 4),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{
						"repository", "artifact", "pull_request",
						"remediate", "alert", "display_name",
					},
				},
				Patch: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace_all",
					Remediate:   proto.String("on"),
					Alert:       proto.String("on"),
					DisplayName: "dsp_test_rem_patch_repo_rule_replace_all",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
						},
					},
					Artifact: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("artifact", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("artifact", 2),
							Def:  &structpb.Struct{},
						},
					},
					PullRequest: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("pull", 2),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("pull", 3),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace_all",
					Remediate:   proto.String("on"),
					Alert:       proto.String("on"),
					DisplayName: "dsp_test_rem_patch_repo_rule_replace_all",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 3),
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 4),
						},
					},
					Artifact: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("artifact", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("artifact", 1),
						},
						{
							Type: ruleTypeName("artifact", 2),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("artifact", 2),
						},
					},
					PullRequest: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("pull", 2),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("pull", 2),
						},
						{
							Type: ruleTypeName("pull", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("pull", 3),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with repo rule replacement",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_patch_repo_rule_replace",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"repository"}},
				Patch: &minderv1.Profile{
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 3),
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 4),
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_rem_patch_repo_rule_replace",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 3),
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 4),
						},
					},
				},
			},
		},
		// Negative tests
		{
			name: "Profile patch does not allow changing name",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_name_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
				Patch: &minderv1.Profile{
					Name: "test_rem_yes_name_change",
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "cannot change profile name",
		},
		{
			name: "Profile patch does not allow changing version",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_version_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"version"}},
				Patch: &minderv1.Profile{
					Version: "2.0.0",
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "profile version is invalid",
		},
		{
			name: "Profile patch does not allow changing type",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_type_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type"}},
				Patch: &minderv1.Profile{
					Type: "not_a_profile",
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "profile type is invalid",
		},
		{
			name: "Profile patch does not allow changing labels",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_label_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				Patch: &minderv1.Profile{
					Labels: []string{"label1", "label2"},
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "labels cannot be updated",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})
			evts := &stubeventer.StubEventer{}
			s := &Server{
				store: dbStore,
				// Do not replace this with a mock - these tests are used to test ProfileService as well
				profiles:      profiles.NewProfileService(evts),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
			}

			// Create the base profile
			baseProfile, err := s.CreateProfile(ctx, tc.baseProfile)
			require.NoError(t, err)

			tc.patchRequest.Id = baseProfile.GetProfile().GetId()
			tc.patchRequest.Context = baseProfile.GetProfile().GetContext()

			patchedProfile, err := s.PatchProfile(ctx, tc.patchRequest)
			if tc.wantErr != "" {
				niceErr, ok := err.(*util.NiceStatus)
				if !ok {
					t.Fatalf("Unexpected error type from PatchProfile: %v", err)
				}
				if !strings.Contains(niceErr.Details, tc.wantErr) {
					t.Errorf("PatchProfile() error = %q, wantErr %q", niceErr.Details, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error from PatchProfile: %v", err)
			}

			profile, err := dbStore.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
				Name:      tc.baseProfile.GetProfile().GetName(),
				ProjectID: dbproj.ID,
			})
			if err != nil {
				t.Fatalf("Error getting profile: %v", err)
			}

			if err != nil {
				t.Fatalf("Error getting profile: %v", err)
			}
			if tc.result.GetProfile().GetId() == "" {
				tc.result.GetProfile().Id = proto.String(profile.ID.String())
			}

			tc.result.GetProfile().Context = tc.baseProfile.GetProfile().GetContext()

			// For some reason, comparing these protos directly doesn't seem to work...
			if !proto.Equal(patchedProfile, tc.result) {
				t.Errorf("PatchProfile() got = %+v\n want %+v", patchedProfile, tc.result)
			}
		})
	}
}

func TestPatchManagedProfile(t *testing.T) {
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

	err = dbStore.UpsertBundle(ctx, db.UpsertBundleParams{
		Name:      "test_managed_profile_bundle",
		Namespace: "testns",
	})
	require.NoError(t, err)

	dbBundle, err := dbStore.GetBundle(ctx, db.GetBundleParams{
		Name:      "test_managed_profile_bundle",
		Namespace: "testns",
	})
	require.NoError(t, err)

	dbSubscription, err := dbStore.CreateSubscription(ctx, db.CreateSubscriptionParams{
		ProjectID: dbproj.ID,
		BundleID:  dbBundle.ID,
	})
	require.NoError(t, err)

	dbProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:           "test_managed_profile",
		ProjectID:      dbproj.ID,
		Alert:          db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
		SubscriptionID: uuid.NullUUID{UUID: dbSubscription.ID, Valid: true},
	})
	require.NoError(t, err)

	ctx = engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{ID: dbproj.ID},
	})
	evts := &stubeventer.StubEventer{}
	s := &Server{
		store: dbStore,
		// Do not replace this with a mock - these tests are used to test ProfileService as well
		profiles:      profiles.NewProfileService(evts),
		providerStore: providers.NewProviderStore(dbStore),
		evt:           evts,
	}

	patchedProfile, err := s.PatchProfile(ctx, &minderv1.PatchProfileRequest{
		Id: dbProfile.ID.String(),
		Patch: &minderv1.Profile{
			Remediate: proto.String("on"),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "attempted to edit a rule type or profile which belongs to a bundle")
	require.Nil(t, patchedProfile)
}

func generateConsistentUUID(t *testing.T, ruleType, ruleName string) uuid.UUID {
	t.Helper()
	return uuid.NewSHA1(uuid.Nil, []byte(ruleType+ruleName))
}
