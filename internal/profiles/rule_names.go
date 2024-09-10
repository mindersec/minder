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
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
