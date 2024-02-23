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
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/profiles"
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

func generateConsistentUUID(t *testing.T, ruleType, ruleName string) uuid.UUID {
	t.Helper()
	return uuid.NewSHA1(uuid.Nil, []byte(ruleType+ruleName))
}
