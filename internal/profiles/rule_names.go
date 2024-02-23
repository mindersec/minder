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
	"github.com/stacklok/minder/internal/engine"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// functions relating to rule names

// ComputeRuleName returns the rule instance's name, or generates a default one
func ComputeRuleName(rule *minderv1.Profile_Rule) string {
	if rule.GetName() != "" {
		return rule.GetName()
	}

	return rule.GetType()
}

// PopulateRuleNames fills in the rule name for all rule instances in a profile
func PopulateRuleNames(profile *minderv1.Profile) {
	_ = engine.TraverseAllRulesForPipeline(profile, func(r *minderv1.Profile_Rule) error {
		r.Name = ComputeRuleName(r)
		return nil
	},
	)
}
