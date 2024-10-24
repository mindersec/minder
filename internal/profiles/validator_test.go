// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/structpb"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/profiles"
	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestValidatorScenarios(t *testing.T) {
	t.Parallel()
	var err error
	// load this data once, since it shared across all tests
	rawRuleDefinition, err := loadRawRuleTypeDef()
	if err != nil {
		t.Fatalf("Could not load test data: %s", err)
	}
	dbReturnsRuleType := dbMockWithRuleType(rawRuleDefinition)

	validatorTestScenarios := []struct {
		Name           string
		Profile        *minderv1.Profile
		DBSetup        func(store *mockdb.MockStore)
		ExpectedError  string
		ExpectedResult profiles.RuleMapping
	}{
		{
			Name:          "Validator rejects profile without mandatory fields",
			Profile:       makeProfile(),
			ExpectedError: "invalid profile",
		},
		{
			Name:          "Validator rejects profile with multiple unnamed rules of same type",
			Profile:       makeProfile(withBasicProfileData, withRules(makeRule(withEmptyRuleName), makeRule(withEmptyRuleName))),
			ExpectedError: "multiple rules with empty name and same type in entity",
		},
		{
			Name:          "Validator rejects profile with multiple rules with same name",
			Profile:       makeProfile(withBasicProfileData, withRules(makeRule(), makeRule())),
			ExpectedError: "multiple rules of same type with same name",
		},
		{
			Name: "Validator rejects profile with multiple rules with same name but different cases",
			Profile: makeProfile(withBasicProfileData, withRules(
				makeRule(withRuleName("myrule")),
				makeRule(withRuleName("MYRULE")),
			)),
			ExpectedError: "multiple rules of same type with same name",
		},
		{
			Name:          "Validator rejects profile with multiple rules with same name and different types",
			Profile:       makeProfile(withBasicProfileData, withRules(makeRule(), makeRule(withRuleType("foo")))),
			ExpectedError: "conflicts with rule name of type",
		},
		{
			Name:          "Validator rejects profile where a rule shares the name of a different type of rule",
			Profile:       makeProfile(withBasicProfileData, withRules(makeRule(withRuleType(ruleName)), makeRule())),
			ExpectedError: "rule name cannot match other rule types",
		},
		{
			Name: "Validator rejects profile where a named rule shares the default name of an empty rule",
			Profile: makeProfile(withBasicProfileData, withRules(
				makeRule(withEmptyRuleName, withRuleType(ruleName)),
				makeRule(withRuleType(ruleName)),
			)),
			ExpectedError: "conflicts with default rule name of unnamed rule",
		},
		{
			Name:          "Validator rejects profile which cannot be found in DB",
			Profile:       makeProfile(withBasicProfileData, withRules(makeRule())),
			DBSetup:       dbReturnsError,
			ExpectedError: "cannot find rule type",
		},
		{
			Name:          "Validator rejects rule instance with missing defs",
			Profile:       makeProfile(withBasicProfileData, withRules(makeRule())),
			DBSetup:       dbReturnsRuleType,
			ExpectedError: "error validating rule",
		},
		{
			Name:          "Validator rejects rule instance with missing params",
			Profile:       makeProfile(withBasicProfileData, withRules(makeRule(withRuleDefs))),
			DBSetup:       dbReturnsRuleType,
			ExpectedError: "error validating rule",
		},
		{
			Name:           "Validator accepts well-formed profile",
			Profile:        makeProfile(withBasicProfileData, withRules(makeRule(withRuleDefs, withRuleParams))),
			DBSetup:        dbReturnsRuleType,
			ExpectedResult: expectation(ruleName, ruleName),
		},
		{
			Name:    "Validator accepts well-formed profile with empty rule name",
			Profile: makeProfile(withBasicProfileData, withRules(makeRule(withRuleDefs, withRuleParams, withEmptyRuleName))),
			DBSetup: dbReturnsRuleType,
			// if rule name is empty in the profile, it should be set to the name of the rule by the validator
			ExpectedResult: expectation("", ruleTypeDisplayName),
		},
		{
			Name: "Validator rejects profile with with rule type that doesn't match entity",
			// Note that this should fail since the default rule definition we have is for a repo entity and not for artifacts
			Profile:       makeProfile(withBasicProfileData, withArtifactRules(makeRule(withRuleDefs, withRuleParams))),
			DBSetup:       dbReturnsRuleType,
			ExpectedError: "expects entity repository, but was given entity artifact",
		},
	}

	for _, testScenario := range validatorTestScenarios {
		t.Run(testScenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			if testScenario.DBSetup != nil {
				testScenario.DBSetup(store)
			}

			v := &profiles.Validator{}
			result, err := v.ValidateAndExtractRules(context.Background(), store, projectID, testScenario.Profile)

			if testScenario.ExpectedError != "" && testScenario.ExpectedResult == nil {
				require.Nil(t, result)
				require.ErrorContains(t, err, testScenario.ExpectedError)
			} else if testScenario.ExpectedError == "" && testScenario.ExpectedResult != nil {
				require.NoError(t, err)
				require.Equal(t, testScenario.ExpectedResult, result)
			} else {
				t.Fatal("Test must define ExpectedError or ExpectedResult, but not both")
			}
		})
	}
}

// fixtures

var ruleTypeName = "branch_protection_allow_force_pushes"
var ruleTypeDisplayName = "Allow force pushes to the branch"
var ruleName = "MyRule"
var ruleUUID = uuid.New()
var projectID = uuid.New()

func withBasicProfileData(profile *minderv1.Profile) {
	profile.Name = "MyProfile"
	profile.Type = minderv1.ProfileType
	profile.Version = minderv1.ProfileTypeVersion
}

// Assumption: for the purposes of unit testing the validation logic, the
// entity types are interchangeable, and we only need to test one type.
func withRules(rules ...*minderv1.Profile_Rule) func(*minderv1.Profile) {
	return func(profile *minderv1.Profile) {
		profile.Repository = rules
	}
}

func withArtifactRules(rules ...*minderv1.Profile_Rule) func(*minderv1.Profile) {
	return func(profile *minderv1.Profile) {
		profile.Artifact = rules
	}
}

func dbReturnsError(store *mockdb.MockStore) {
	store.EXPECT().
		GetParentProjects(gomock.Any(), gomock.Any()).
		Return([]uuid.UUID{uuid.New()}, nil).
		AnyTimes()
	store.EXPECT().
		GetRuleTypeByName(gomock.Any(), gomock.Any()).
		Return(db.RuleType{}, sql.ErrNoRows).
		AnyTimes()
}

func dbMockWithRuleType(rawRuleDefinition json.RawMessage) func(*mockdb.MockStore) {
	return func(store *mockdb.MockStore) {
		ruleType := db.RuleType{
			ID:          ruleUUID,
			Name:        ruleTypeName,
			DisplayName: ruleTypeDisplayName,
			Definition:  rawRuleDefinition,
		}

		store.EXPECT().
			GetParentProjects(gomock.Any(), gomock.Any()).
			Return([]uuid.UUID{uuid.New()}, nil).
			AnyTimes()
		store.EXPECT().
			GetRuleTypeByName(gomock.Any(), gomock.Any()).
			Return(ruleType, nil).
			AnyTimes()
	}
}

func expectation(profileRuleName string, expectedRuleName string) profiles.RuleMapping {
	return profiles.RuleMapping{
		profiles.RuleTypeAndNamePair{
			RuleType: ruleTypeName,
			RuleName: profileRuleName,
		}: profiles.RuleIdAndNamePair{
			RuleID:          ruleUUID,
			DerivedRuleName: expectedRuleName,
		},
	}
}

func makeRule(opts ...func(rule *minderv1.Profile_Rule)) *minderv1.Profile_Rule {
	rule := &minderv1.Profile_Rule{
		Type:   ruleTypeName,
		Name:   ruleName,
		Def:    &structpb.Struct{},
		Params: &structpb.Struct{},
	}
	for _, opt := range opts {
		opt(rule)
	}
	return rule
}

func withEmptyRuleName(rule *minderv1.Profile_Rule) {
	rule.Name = ""
}

func withRuleName(name string) func(rule *minderv1.Profile_Rule) {
	return func(rule *minderv1.Profile_Rule) {
		rule.Name = name
	}
}

func withRuleType(typeName string) func(rule *minderv1.Profile_Rule) {
	return func(rule *minderv1.Profile_Rule) {
		rule.Type = typeName
	}
}

func withRuleParams(rule *minderv1.Profile_Rule) {
	rule.Params.Fields = map[string]*structpb.Value{
		"branch": {
			Kind: &structpb.Value_StringValue{
				StringValue: "main",
			},
		},
	}
}

func withRuleDefs(rule *minderv1.Profile_Rule) {
	rule.Def.Fields = map[string]*structpb.Value{
		"allow_force_pushes": {
			Kind: &structpb.Value_BoolValue{
				BoolValue: false,
			},
		},
	}
}

func makeProfile(opts ...func(*minderv1.Profile)) *minderv1.Profile {
	profile := &minderv1.Profile{}
	for _, opt := range opts {
		opt(profile)
	}
	return profile
}

func loadRawRuleTypeDef() (json.RawMessage, error) {
	// read rule type from disk and set it up as a fixture
	f, err := os.Open("../../examples/rules-and-profiles/rule-types/github/branch_protection_allow_force_pushes.yaml")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ruleType := &minderv1.RuleType{}
	if err := minderv1.ParseResource(f, ruleType); err != nil {
		return nil, err
	}

	raw, err := util.GetBytesFromProto(ruleType.GetDef())
	if err != nil {
		return nil, err
	}
	return raw, nil
}
