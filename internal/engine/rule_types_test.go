// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stacklok/mediator/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestExampleRulesAreValidatedCorrectly(t *testing.T) {
	t.Parallel()

	t.Log("parsing example policy")
	pol, err := engine.ReadPolicyFromFile("../../examples/github/policies/policy.yaml")
	require.NoError(t, err, "failed to parse example policy")

	// open rules in example directory
	filepath.Walk("../../examples/github/rule-types", func(path string, info os.FileInfo, err error) error {
		// skip directories
		if info.IsDir() {
			return nil
		}

		// skip non-yaml files
		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		fname := filepath.Base(path)
		t.Run(fname, func(t *testing.T) {
			t.Parallel()

			// open file
			f, err := os.Open(path)
			require.NoError(t, err, "failed to open file %s", path)
			defer f.Close()

			t.Log("parsing rule type", path)
			rt, err := engine.ParseRuleType(f)
			require.NoError(t, err, "failed to parse rule type %s", path)
			require.NotNil(t, rt, "failed to parse rule type %s", path)

			t.Log("creating rule validator")
			rval, err := engine.NewRuleValidator(rt)
			require.NoError(t, err, "failed to create rule validator for rule type %s", path)

			rules, err := engine.GetRulesFromPolicyOfType(pol, rt)
			require.NoError(t, err, "failed to get rules from policy for rule type %s", path)

			t.Log("validating rules")
			for _, ruleCall := range rules {
				err := rval.ValidateRuleDefAgainstSchema(ruleCall.Def.AsMap())
				require.NoError(t, err, "failed to validate rule definition for rule type %s", path)

				err = rval.ValidateParamsAgainstSchema(ruleCall.GetParams())
				require.NoError(t, err, "failed to validate rule parameters for rule type %s", path)
			}

		})

		return nil
	})
}
