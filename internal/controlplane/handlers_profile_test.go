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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/engine"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestValidateRuleNameAndType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entity     minderv1.Entity
		rules      []*minderv1.Profile_Rule
		wantErrMsg string
	}{
		{
			name:   "Valid rule names and types",
			entity: minderv1.Entity_ENTITY_REPOSITORIES,
			rules: []*minderv1.Profile_Rule{
				{Name: "rule1", Type: "type1"},
				{Name: "rule2", Type: "type2"},
			},
		},
		{
			name:   "Duplicate rule names with different types",
			entity: minderv1.Entity_ENTITY_REPOSITORIES,
			rules: []*minderv1.Profile_Rule{
				{Name: "rule1", Type: "type1"},
				{Name: "rule1", Type: "type2"},
			},
			wantErrMsg: "profile contained invalid rule 'type2': rule name 'rule1' conflicts with rule name of type 'type1' in entity 'repository', assign unique names to rules",
		},
		{
			name:   "Rule with same name as it's rule type",
			entity: minderv1.Entity_ENTITY_REPOSITORIES,
			rules: []*minderv1.Profile_Rule{
				{Name: "rule1", Type: "type1"},
				{Name: "type1", Type: "type1"},
			},
		},
		{
			name:   "Rule with same name as it's rule type and other rule with no name, same type",
			entity: minderv1.Entity_ENTITY_REPOSITORIES,
			rules: []*minderv1.Profile_Rule{
				{Name: "type1", Type: "type1"},
				{Name: "", Type: "type1"},
			},
			wantErrMsg: "profile contained invalid rule 'type1': rule name 'type1' conflicts with default rule name of unnamed rule in entity 'repository', assign unique names to rules",
		},
		{
			name:   "Duplicate rule names with same types",
			entity: minderv1.Entity_ENTITY_REPOSITORIES,
			rules: []*minderv1.Profile_Rule{
				{Name: "rule1", Type: "type1"},
				{Name: "rule1", Type: "type1"},
			},
			wantErrMsg: "profile contained invalid rule 'type1': multiple rules of same type with same name 'rule1' in entity 'repository', assign unique names to rules",
		},
		{
			name:   "Empty rule names with same types",
			entity: minderv1.Entity_ENTITY_REPOSITORIES,
			rules: []*minderv1.Profile_Rule{
				{Name: "", Type: "type1"},
				{Name: "", Type: "type1"},
			},
			wantErrMsg: "profile contained invalid rule 'type1': multiple rules with empty name and same type in entity 'repository', add unique names to rules",
		},
		{
			name:   "Multiple rules with empty names and different types",
			entity: minderv1.Entity_ENTITY_ARTIFACTS,
			rules: []*minderv1.Profile_Rule{
				{Name: "", Type: "type1"},
				{Name: "", Type: "type2"},
				{Name: "some name", Type: "type2"},
				{Name: "", Type: "type1"},
			},
			wantErrMsg: "profile contained invalid rule 'type1': multiple rules with empty name and same type in entity 'artifact', add unique names to rules",
		},
		{
			name:   "Multiple rules with empty names and same types",
			entity: minderv1.Entity_ENTITY_ARTIFACTS,
			rules: []*minderv1.Profile_Rule{
				{Name: "", Type: "type1"},
				{Name: "some name 1", Type: "type1"},
				{Name: "some name 2", Type: "type1"},
				{Name: "some name 3", Type: "type1"},
				{Name: "", Type: "type1"},
			},
			wantErrMsg: "profile contained invalid rule 'type1': multiple rules with empty name and same type in entity 'artifact', add unique names to rules",
		},
		{
			name:   "Multiple rules of same type but different names",
			entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS,
			rules: []*minderv1.Profile_Rule{
				{Name: "", Type: "type1"},
				{Name: "rule1", Type: "type1"},
				{Name: "rule2", Type: "type1"},
				{Name: "rule3", Type: "type1"},
				{Name: "", Type: "type1"},
				{Name: "", Type: "type2"},
				{Name: "", Type: "type3"},
			},
			wantErrMsg: "profile contained invalid rule 'type1': multiple rules with empty name and same type in entity 'build_environment', add unique names to rules",
		},
		{
			name:   "Rule with name same as other rule type",
			entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS,
			rules: []*minderv1.Profile_Rule{
				{Name: "", Type: "type1"},
				{Name: "", Type: "type2"},
				{Name: "type1", Type: "type3"},
			},
			wantErrMsg: "profile contained invalid rule 'type3': rule name 'type1' conflicts with a rule type in entity 'build_environment', rule name cannot match other rule types",
		},
		{
			name:   "Rule name same as default rule name",
			entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS,
			rules: []*minderv1.Profile_Rule{
				{Name: "", Type: "type1"},
				{Name: "", Type: "type2"},
				{Name: "rule1", Type: "type3"},
				{Name: "type1", Type: "type1"},
			},
			wantErrMsg: "profile contained invalid rule 'type1': rule name 'type1' conflicts with default rule name of unnamed rule in entity 'build_environment', assign unique names to rules",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateRuleNameAndType(test.entity, test.rules)

			if test.wantErrMsg != "" {
				require.Error(t, err)

				var v *engine.RuleValidationError
				require.True(t, errors.As(err, &v))
				errMsg := fmt.Sprintf("profile contained invalid rule '%s': %s", v.RuleType, v.Err)
				require.Equal(t, test.wantErrMsg, errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetUnusedOldRuleTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		newRules   map[ruleTypeAndNamePair]entityAndRuleTuple
		oldRules   map[ruleTypeAndNamePair]entityAndRuleTuple
		wantUnused []entityAndRuleTuple
	}{
		{
			name: "Unused rule in oldRules",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
			wantUnused: []entityAndRuleTuple{
				{Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
		},
		{
			name: "Multiple unused rules in oldRules",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
				{RuleType: "Type4", RuleName: "Name4"}: {Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type4", "Name4")},
			},
			wantUnused: []entityAndRuleTuple{
				{Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
				{Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type4", "Name4")},
			},
		},
		{
			name: "No unused rules in oldRules",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			wantUnused: nil,
		},
		{
			name: "Unused rules with same rule type",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "Name3")},
			},
			// All rule types are used
			wantUnused: nil,
		},
		{
			name: "Unused rules with same rule type but different entity types",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
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
		newRules        map[ruleTypeAndNamePair]entityAndRuleTuple
		oldRules        map[ruleTypeAndNamePair]entityAndRuleTuple
		wantUnusedRules map[ruleTypeAndNamePair]entityAndRuleTuple
	}{
		{
			name: "Unused rule in oldRules",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
			wantUnusedRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type3", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type3", "Name3")},
			},
		},
		{
			name: "No unused rules in oldRules",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type2", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type2", "Name2")},
			},
			wantUnusedRules: map[ruleTypeAndNamePair]entityAndRuleTuple{},
		},
		{
			name: "Unused rules with same rule type",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "Name3")},
			},
			wantUnusedRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name3"}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "Name3")},
			},
		},
		{
			name: "Unused old rules statuses with empty name",
			newRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: "Name1"}: {Entity: minderv1.Entity_ENTITY_REPOSITORIES, RuleID: generateConsistentUUID(t, "Type1", "Name1")},
				{RuleType: "Type1", RuleName: "Name2"}: {Entity: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, RuleID: generateConsistentUUID(t, "Type1", "Name2")},
			},
			oldRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
				{RuleType: "Type1", RuleName: ""}: {Entity: minderv1.Entity_ENTITY_ARTIFACTS, RuleID: generateConsistentUUID(t, "Type1", "")},
				{RuleType: "Type2", RuleName: ""}: {Entity: minderv1.Entity_ENTITY_PULL_REQUESTS, RuleID: generateConsistentUUID(t, "Type2", "")},
			},
			wantUnusedRules: map[ruleTypeAndNamePair]entityAndRuleTuple{
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
