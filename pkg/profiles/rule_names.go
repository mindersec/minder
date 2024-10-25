// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// functions relating to rule names

// ComputeRuleName returns the rule instance's name, or generates a default one
func ComputeRuleName(rule *minderv1.Profile_Rule, ruleTypeDisplayName string) string {
	if rule.GetName() != "" {
		return rule.GetName()
	}

	return ruleTypeDisplayName
}

// PopulateRuleNames fills in the rule name for all rule instances in a profile
func PopulateRuleNames(profile *minderv1.Profile, rules RuleMapping) {
	_ = TraverseAllRulesForPipeline(profile, func(r *minderv1.Profile_Rule) error {
		key := RuleTypeAndNamePair{
			RuleType: r.GetType(),
			RuleName: r.GetName(),
		}
		value := rules[key]
		r.Name = value.DerivedRuleName

		// update the rule name in the rules map
		newKey := RuleTypeAndNamePair{
			RuleType: r.GetType(),
			RuleName: value.DerivedRuleName,
		}
		delete(rules, key)
		rules[newKey] = value

		return nil
	},
	)
}
