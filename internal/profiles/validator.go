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

package profiles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/selectors"
	"github.com/stacklok/minder/internal/ruletypes"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Validator encapsulates the logic for validating profiles
type Validator struct {
	selBld selectors.SelectionChecker
}

// NewValidator creates a new profile validator
func NewValidator(selBld selectors.SelectionChecker) *Validator {
	return &Validator{selBld: selBld}
}

// ValidateAndExtractRules validates a profile to ensure it is well-formed
// it also returns information about the rules in the profile
func (v *Validator) ValidateAndExtractRules(
	ctx context.Context,
	qtx db.Querier,
	projectID uuid.UUID,
	profile *minderv1.Profile,
) (RuleMapping, error) {

	// ensure that the profile has all required fields
	if err := profile.Validate(); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile: %v", err)
	}

	// ensure that the rule names follow all naming constraints
	if err := validateRuleNames(profile); err != nil {
		var violation *RuleValidationError
		if errors.As(err, &violation) {
			return nil, util.UserVisibleError(codes.InvalidArgument,
				"profile failed rule name validation: %s", violation)
		}
		return nil, status.Errorf(codes.Internal, "error validating rule names in profile")
	}

	// validate that the rule invocations match what the rule types expect
	if err := validateEntities(ctx, qtx, profile, projectID); err != nil {
		var violation *RuleValidationError
		if errors.As(err, &violation) {
			return nil, util.UserVisibleError(codes.InvalidArgument,
				"profile failed rule validation: %s", violation)
		}
		return nil, status.Errorf(codes.Internal, "error validating profile")
	}

	// validate that the parameters for the rules match the expected schema
	rulesInProf, err := v.validateRuleParams(ctx, qtx, profile, projectID)
	if err != nil {
		var violation *RuleValidationError
		if errors.As(err, &violation) {
			log.Printf("error validating rule: %v", violation)
			return nil, util.UserVisibleError(codes.InvalidArgument,
				"profile contained invalid rule '%s': %s", violation.RuleType, violation.Err)
		}

		log.Printf("error getting rule type: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	// once validated, return the list of all rules for this profile
	return rulesInProf, nil
}

func (_ *Validator) validateRuleParams(
	ctx context.Context,
	qtx db.Querier,
	prof *minderv1.Profile,
	projectID uuid.UUID,
) (RuleMapping, error) {
	// We capture the rule instantiations here, so we can
	// track them in the db later.
	rulesInProfile := make(RuleMapping)

	// Allows a profile to use rules from parent projects
	projects, err := qtx.GetParentProjects(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error getting parent projects: %w", err)
	}

	err = TraverseAllRulesForPipeline(prof, func(profileRule *minderv1.Profile_Rule) error {
		ruleType, err := qtx.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Projects: projects,
			Name:     profileRule.GetType(),
		})

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return &RuleValidationError{
					Err:      fmt.Sprintf("cannot find rule type %s", profileRule.GetType()),
					RuleType: profileRule.GetType(),
				}
			}

			return fmt.Errorf("error getting rule type %s: %w", profileRule.GetType(), err)
		}

		ruleTypePB, err := ruletypes.RuleTypePBFromDB(&ruleType)
		if err != nil {
			return fmt.Errorf("cannot convert rule type %s to minderv1: %w", ruleType.Name, err)
		}

		ruleValidator, err := NewRuleValidator(ruleTypePB)
		if err != nil {
			return fmt.Errorf("error creating rule validator: %w", err)
		}

		if err := ruleValidator.ValidateRuleDefAgainstSchema(profileRule.Def.AsMap()); err != nil {
			return fmt.Errorf("error validating rule: %w", err)
		}

		if err := ruleValidator.ValidateParamsAgainstSchema(profileRule.GetParams()); err != nil {
			return fmt.Errorf("error validating rule params: %w", err)
		}

		ruleName := ComputeRuleName(profileRule)

		key := RuleTypeAndNamePair{
			RuleType: profileRule.GetType(),
			RuleName: ruleName,
		}

		rulesInProfile[key] = EntityAndRuleTuple{
			Entity: minderv1.EntityFromString(ruleTypePB.Def.InEntity),
			RuleID: ruleType.ID,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return rulesInProfile, nil
}

func validateRuleNames(profile *minderv1.Profile) error {
	for ent, entRules := range map[minderv1.Entity][]*minderv1.Profile_Rule{
		minderv1.Entity_ENTITY_REPOSITORIES:       profile.GetRepository(),
		minderv1.Entity_ENTITY_ARTIFACTS:          profile.GetArtifact(),
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS: profile.GetBuildEnvironment(),
		minderv1.Entity_ENTITY_PULL_REQUESTS:      profile.GetPullRequest(),
		minderv1.Entity_ENTITY_RELEASE:            profile.GetRelease(),
		minderv1.Entity_ENTITY_PIPELINE_RUN:       profile.GetPipelineRun(),
		minderv1.Entity_ENTITY_TASK_RUN:           profile.GetTaskRun(),
		minderv1.Entity_ENTITY_BUILD:              profile.GetBuild(),
	} {
		if err := validateRuleNamesForEntity(ent, entRules); err != nil {
			return err
		}
	}

	return nil
}

// validateRuleNamesForEntity validates that the rules in the profile have unique names and types.
// Default Rule Name: For rules with no name, rule type is assumed to be the rule name.
// Validation rules:
// 1. Rule name can't match other rule types (excluding default rule name)
// 2. Rule name can't be empty if there are multiple rules with no name and same type
// 3. Non-empty rule name can't match any other rule name (including default rule name)
func validateRuleNamesForEntity(entity minderv1.Entity, rules []*minderv1.Profile_Rule) error {
	ruleNameToType := make(map[string]string)

	typesSet := sets.New[string]()
	emptyNameTypesSet := sets.New[string]()

	for _, rule := range rules {
		ruleName := rule.GetName()
		ruleType := rule.GetType()
		typesSet.Insert(ruleType)

		if typesSet.Has(ruleName) && ruleName != ruleType {
			return &RuleValidationError{
				Err: fmt.Sprintf("rule name '%s' conflicts with a rule type in entity '%s', rule name cannot match other rule types",
					ruleName, entity.ToString()),
				RuleType: ruleType,
			}
		}

		if ruleName == "" {
			err := validateRuleWithEmptyName(ruleType, entity, emptyNameTypesSet)
			if err != nil {
				return err
			}
		}
	}

	for _, rule := range rules {
		if rule.GetName() != "" {
			err := validateRuleWithNonEmptyName(rule, entity, ruleNameToType, emptyNameTypesSet)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRuleWithEmptyName(
	ruleType string,
	entity minderv1.Entity,
	emptyNameTypesSet sets.Set[string],
) error {
	if emptyNameTypesSet.Has(ruleType) {
		return &RuleValidationError{
			Err: fmt.Sprintf(
				"multiple rules with empty name and same type in entity '%s', add unique names to rules", entity.ToString()),
			RuleType: ruleType,
		}
	}
	emptyNameTypesSet.Insert(ruleType)
	return nil
}

func validateRuleWithNonEmptyName(
	rule *minderv1.Profile_Rule, entity minderv1.Entity,
	ruleNameToType map[string]string, emptyNameTypesSet sets.Set[string],
) error {
	ruleName := rule.GetName()
	ruleType := rule.GetType()
	if existingType, ok := ruleNameToType[ruleName]; ok {
		if existingType == ruleType {
			return &RuleValidationError{
				Err: fmt.Sprintf("multiple rules of same type with same name '%s' in entity '%s', assign unique names to rules",
					ruleName, entity.ToString()),
				RuleType: ruleType,
			}
		}
		return &RuleValidationError{
			Err: fmt.Sprintf("rule name '%s' conflicts with rule name of type '%s' in entity '%s', assign unique names to rules",
				ruleName, existingType, entity.ToString()),
			RuleType: ruleType,
		}

	}

	if ruleName == ruleType && emptyNameTypesSet.Has(ruleType) {
		return &RuleValidationError{
			Err: fmt.Sprintf(
				"rule name '%s' conflicts with default rule name of unnamed rule in entity '%s', assign unique names to rules",
				ruleName, entity.ToString()),
			RuleType: ruleType,
		}
	}

	ruleNameToType[ruleName] = ruleType
	return nil
}

func validateEntities(
	ctx context.Context,
	qtx db.Querier,
	profile *minderv1.Profile,
	projectID uuid.UUID,
) error {
	projects, err := qtx.GetParentProjects(ctx, projectID)
	if err != nil {
		return fmt.Errorf("error getting parent projects: %w", err)
	}

	// validate that the entities in the profile match the entities in the project
	err = TraverseRuleTypesForEntities(profile, func(entity minderv1.Entity, rule *minderv1.Profile_Rule) error {
		ruleType, err := qtx.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Projects: projects,
			Name:     rule.GetType(),
		})

		if err != nil {
			// This is checked elsewhere. We probably want to converge and
			// do a single check, but that would require a wider refactor.
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}

			return fmt.Errorf("error getting rule type %s: %w", rule.GetType(), err)
		}

		ruleTypePB, err := ruletypes.RuleTypePBFromDB(&ruleType)
		if err != nil {
			return fmt.Errorf("cannot convert rule type %s to minderv1: %w", ruleType.Name, err)
		}

		if ruleTypePB.Def.InEntity != entity.ToString() {
			return &RuleValidationError{
				Err: fmt.Sprintf("rule type %s expects entity %s, but was given entity %s",
					ruleTypePB.Name, ruleTypePB.Def.InEntity, entity.ToString()),
				RuleType: ruleTypePB.Name,
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// ValidateSelection validates the selectors in a profile
func (v *Validator) ValidateSelection(
	selection []*minderv1.Profile_Selector,
) error {
	for _, sel := range selection {
		if err := v.selBld.CheckSelector(sel); err != nil {
			return handleSelectorError(err)
		}
	}

	return nil
}

func handleSelectorError(err error) error {
	if errors.Is(err, selectors.ErrUnsupported) {
		return util.UserVisibleError(codes.InvalidArgument, "unsupported entity type: %v", err)
	} else if errors.Is(err, selectors.ErrSelectorCheck) {
		return handleErrSelectorCheck(err)
	}

	// fallback, just return the error as-is
	return err
}

func handleErrSelectorCheck(err error) error {
	var selParseErr *selectors.ParseError
	var selCheckErr *selectors.CheckError

	if errors.As(err, &selParseErr) {
		msg := "No error message found"
		if len(selParseErr.Errors) > 0 {
			msg = selParseErr.Errors[0].Msg
		}
		return util.UserVisibleError(codes.InvalidArgument, "selector failed to parse: %s", msg)
	} else if errors.As(err, &selCheckErr) {
		msg := "No error message found"
		if len(selCheckErr.Errors) > 0 {
			msg = selCheckErr.Errors[0].Msg
		}
		return util.UserVisibleError(codes.InvalidArgument, "selector is invalid: %s", msg)
	}

	return err
}
